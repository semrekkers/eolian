package module

import (
	"fmt"
	"math/rand"

	"buddin.us/eolian/dsp"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("StageSequence", func(c Config) (Patcher, error) {
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
	multiOutIO

	clock, transpose, reset, glide, mode *In
	stages                               []stage
	pulse, stage, lastStage              int
	pong                                 bool
	slew                                 *slew

	lastClock, lastReset, rollingVelocity dsp.Float64

	gateOut,
	pitchOut,
	velocityOut,
	syncOut,
	endStageOut dsp.Frame
}

type stage struct {
	pitch, pulses, gateMode, glide, velocity *In
}

func newStageSequence(stages int) (*stageSequence, error) {
	m := &stageSequence{
		clock:       NewInBuffer("clock", dsp.Float64(0)),
		transpose:   NewInBuffer("transpose", dsp.Float64(1)),
		reset:       NewInBuffer("reset", dsp.Float64(0)),
		glide:       NewInBuffer("glide", dsp.Float64(0)),
		mode:        NewInBuffer("mode", dsp.Float64(0)),
		stages:      make([]stage, stages),
		lastClock:   -1,
		lastReset:   -1,
		lastStage:   -1,
		pulse:       -1,
		slew:        newSlew(),
		gateOut:     dsp.NewFrame(),
		pitchOut:    dsp.NewFrame(),
		velocityOut: dsp.NewFrame(),
		syncOut:     dsp.NewFrame(),
		endStageOut: dsp.NewFrame(),
	}

	inputs := []*In{m.clock, m.transpose, m.reset, m.glide, m.mode}

	for i := 0; i < stages; i++ {
		m.stages[i] = stage{
			pitch:    NewInBuffer(fmt.Sprintf("%d/pitch", i), dsp.Float64(0)),
			pulses:   NewInBuffer(fmt.Sprintf("%d/pulses", i), dsp.Float64(1)),
			gateMode: NewInBuffer(fmt.Sprintf("%d/mode", i), dsp.Float64(1)),
			glide:    NewInBuffer(fmt.Sprintf("%d/glide", i), dsp.Float64(0)),
			velocity: NewInBuffer(fmt.Sprintf("%d/velocity", i), dsp.Float64(1)),
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
			{Name: "gate", Provider: provideCopyOut(m, &m.gateOut)},
			{Name: "pitch", Provider: provideCopyOut(m, &m.pitchOut)},
			{Name: "velocity", Provider: provideCopyOut(m, &m.velocityOut)},
			{Name: "endstage", Provider: provideCopyOut(m, &m.endStageOut)},
			{Name: "sync", Provider: provideCopyOut(m, &m.syncOut)},
		},
	)
}

func (s *stageSequence) Process(out dsp.Frame) {
	s.incrRead(func() {

		clock := s.clock.ProcessFrame()
		reset := s.reset.ProcessFrame()
		mode := s.mode.ProcessFrame()
		transpose := s.transpose.ProcessFrame()
		glide := s.glide.ProcessFrame()

		for _, stg := range s.stages {
			stg.pitch.ProcessFrame()
			stg.pulses.ProcessFrame()
			stg.gateMode.ProcessFrame()
			stg.glide.ProcessFrame()
			stg.velocity.ProcessFrame()
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
			s.fillEndStage(i)
			s.fillVelocity(i)

			s.lastClock = clock[i]
			s.lastReset = reset[i]
		}
	})
}

func (s *stageSequence) fillGate(i int, clock dsp.Float64) {
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

func (s *stageSequence) fillPitch(i int, transpose, glideAmount dsp.Float64) {
	stage := s.stages[s.stage]
	in := stage.pitch.LastFrame()[i] * transpose
	glide := stage.glide.LastFrame()[i]
	if glide > 0 {
		s.pitchOut[i] = s.slew.Tick(in, glideAmount, glideAmount)
	} else {
		s.pitchOut[i] = in
	}
}

func (s *stageSequence) fillEndStage(i int) {
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

func mapPatternMode(v dsp.Float64) int {
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

func mapGateMode(v dsp.Float64) int {
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
