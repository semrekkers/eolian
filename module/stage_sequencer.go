package module

import (
	"fmt"
	"math/rand"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Sequence", func(c Config) (Patcher, error) {
		var config struct {
			Stages int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Stages == 0 {
			config.Stages = 8
		}
		return newStageSequence(config.Stages)
	})
}

const (
	gateModeRest int = iota
	gateModeSingle
	gateModeRepeat
	gateModeHold

	patternModeSequential int = iota
	patternModePingPong
	patternModeRandom
)

type stageSequence struct {
	IO
	readTracker manyReadTracker

	clock, transpose, reset, glide, mode *In
	stages                               []stage
	pulse, stage, lastStage              int
	pong                                 bool
	slew                                 *slew

	lastClock, lastReset, rollingVelocity Value

	gateOut,
	pitchOut,
	velocityOut,
	syncOut,
	endStageOut Frame
}

type stage struct {
	pitch, pulses, gateMode, glide, velocity *In
}

func newStageSequence(stages int) (*stageSequence, error) {
	m := &stageSequence{
		clock:       &In{Name: "clock", Source: NewBuffer(zero)},
		transpose:   &In{Name: "transpose", Source: NewBuffer(Value(1))},
		reset:       &In{Name: "reset", Source: NewBuffer(zero)},
		glide:       &In{Name: "glide", Source: NewBuffer(zero)},
		mode:        &In{Name: "mode", Source: NewBuffer(zero)},
		stages:      make([]stage, stages),
		lastClock:   -1,
		lastReset:   -1,
		lastStage:   -1,
		pulse:       -1,
		slew:        newSlew(),
		gateOut:     make(Frame, FrameSize),
		pitchOut:    make(Frame, FrameSize),
		velocityOut: make(Frame, FrameSize),
		syncOut:     make(Frame, FrameSize),
		endStageOut: make(Frame, FrameSize),
	}
	m.readTracker = manyReadTracker{counter: m}

	inputs := []*In{m.clock, m.transpose, m.reset, m.glide, m.mode}

	for i := 0; i < stages; i++ {
		m.stages[i] = stage{
			pitch: &In{
				Name:   fmt.Sprintf("%d/pitch", i),
				Source: NewBuffer(zero),
			},
			pulses: &In{
				Name:   fmt.Sprintf("%d/pulses", i),
				Source: NewBuffer(Value(1)),
			},
			gateMode: &In{
				Name:   fmt.Sprintf("%d/mode", i),
				Source: NewBuffer(Value(1)),
			},
			glide: &In{
				Name:   fmt.Sprintf("%d/glide", i),
				Source: NewBuffer(zero),
			},
			velocity: &In{
				Name:   fmt.Sprintf("%d/velocity", i),
				Source: NewBuffer(Value(1)),
			},
		}
		inputs = append(inputs,
			m.stages[i].pitch,
			m.stages[i].pulses,
			m.stages[i].gateMode,
			m.stages[i].glide,
			m.stages[i].velocity)
	}

	return m, m.Expose(
		"StageSequence",
		inputs,
		[]*Out{
			{Name: "gate", Provider: m.out(&m.gateOut)},
			{Name: "pitch", Provider: m.out(&m.pitchOut)},
			{Name: "velocity", Provider: m.out(&m.velocityOut)},
			{Name: "endstage", Provider: m.out(&m.endStageOut)},
			{Name: "sync", Provider: m.out(&m.syncOut)},
		},
	)
}

func (s *stageSequence) out(cache *Frame) ReaderProvider {
	return ReaderProviderFunc(func() Reader {
		return &manyOut{reader: s, cache: cache}
	})
}

func (s *stageSequence) readMany(out Frame) {
	if s.readTracker.count() > 0 {
		s.readTracker.incr()
		return
	}

	clock := s.clock.ReadFrame()
	reset := s.reset.ReadFrame()
	mode := s.mode.ReadFrame()
	transpose := s.transpose.ReadFrame()
	glide := s.glide.ReadFrame()

	for _, stg := range s.stages {
		stg.pitch.ReadFrame()
		stg.pulses.ReadFrame()
		stg.gateMode.ReadFrame()
		stg.glide.ReadFrame()
		stg.velocity.ReadFrame()
	}

	for i := range out {
		if s.lastClock < 0 && clock[i] > 0 {
			pulses := s.stages[s.stage].pulses.LastFrame()[i]
			lastPulse := s.pulse
			s.pulse = (s.pulse + 1) % int(pulses)

			if lastPulse >= 0 && s.pulse == 0 {
				s.lastStage = s.stage
				switch mapPatternMode(mode[i]) {
				case patternModeSequential:
					s.stage = (s.stage + 1) % len(s.stages)
					s.pong = false
				case patternModePingPong:
					var inc = 1
					if s.pong {
						inc = -1
					}
					s.stage += inc

					if s.stage == len(s.stages)-1 {
						s.stage = len(s.stages) - 1
						s.pong = true
					} else if s.stage == 0 {
						s.stage = 0
						s.pong = false
					}
				case patternModeRandom:
					s.stage = rand.Intn(len(s.stages))
					s.pong = false
				}
			}
		}
		if s.lastReset < 0 && reset[0] > 0 {
			s.pulse = 0
			s.stage = 0
		}

		s.fillGate(i, clock[i])
		s.fillPitch(i, transpose[i], glide[i])
		s.fillEndOfStage(i)
		s.fillVelocity(i)

		s.lastClock = clock[i]
		s.lastReset = reset[i]
	}
	s.readTracker.incr()
}

func (s *stageSequence) fillGate(i int, clock Value) {
	gateMode := s.stages[s.stage].gateMode.LastFrame()

	switch mapGateMode(gateMode[i]) {
	case gateModeHold:
		s.gateOut[i] = 1
	case gateModeRepeat:
		if clock > 0 {
			s.gateOut[i] = 1
		} else {
			s.gateOut[i] = -1
		}
	case gateModeSingle:
		if s.pulse == 0 && clock > 0 {
			s.gateOut[i] = 1
		} else {
			s.gateOut[i] = -1
		}
	case gateModeRest:
		s.gateOut[i] = -1
	}
}

func (s *stageSequence) fillPitch(i int, transpose, glideAmount Value) {
	stage := s.stages[s.stage]
	in := stage.pitch.LastFrame()[i] * transpose
	glide := stage.glide.LastFrame()[i]
	if glide > 0 {
		s.pitchOut[i] = s.slew.Tick(in, glideAmount, glideAmount)
	} else {
		s.pitchOut[i] = in
	}
}

func (s *stageSequence) fillEndOfStage(i int) {
	if s.lastStage != s.stage {
		s.endStageOut[i] = 1
	} else {
		s.endStageOut[i] = -1
	}
}

func (s *stageSequence) fillVelocity(i int) {
	velocity := s.stages[s.stage].velocity.LastFrame()
	s.rollingVelocity -= s.rollingVelocity / averageVelocitySamples
	s.rollingVelocity += velocity[i] / averageVelocitySamples
	s.velocityOut[i] = s.rollingVelocity
}

const averageVelocitySamples = 100

func mapPatternMode(v Value) int {
	switch v {
	case 0:
		return patternModeSequential
	case 1:
		return patternModePingPong
	case 2:
		return patternModeRandom
	default:
		return patternModeRandom
	}
}

func mapGateMode(v Value) int {
	switch int(v) {
	case 0:
		return gateModeRest
	case 1:
		return gateModeSingle
	case 2:
		return gateModeRepeat
	case 3:
		return gateModeHold
	default:
		return gateModeHold
	}
}
