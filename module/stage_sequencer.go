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
	clock, transpose, reset, glide, mode *In
	stages                               []stage
	reads                                int
	stage, lastStage                     int
	pulse                                int
	gateMode                             int
	lastClock, lastReset                 Value
	pong                                 bool
}

type stage struct {
	pitch, pulses, gateMode, glide, velocity *In
}

func newStageSequence(stages int) (*stageSequence, error) {
	m := &stageSequence{
		clock:     &In{Name: "clock", Source: NewBuffer(zero)},
		transpose: &In{Name: "transpose", Source: NewBuffer(Value(1))},
		reset:     &In{Name: "reset", Source: NewBuffer(zero)},
		glide:     &In{Name: "glide", Source: NewBuffer(zero)},
		mode:      &In{Name: "mode", Source: NewBuffer(zero)},
		stages:    make([]stage, stages),
		lastClock: -1,
		lastReset: -1,
		lastStage: -1,
		pulse:     -1,
	}

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
			{Name: "gate", Provider: Provide(&stageSeqGate{stageSequence: m})},
			{Name: "pitch", Provider: Provide(&stageSeqPitch{stageSequence: m, slew: newSlew()})},
			{Name: "velocity", Provider: Provide(&stageSeqVelocity{stageSequence: m})},
			{Name: "endstage", Provider: Provide(&stageSeqEndStage{stageSequence: m})},
			{Name: "sync", Provider: Provide(&stageSeqSync{stageSequence: m})},
		},
	)
}

func (s *stageSequence) read(out Frame) {
	if s.reads > 0 {
		return
	}

	s.clock.ReadFrame()
	s.reset.ReadFrame()
	s.mode.ReadFrame()
	s.transpose.ReadFrame()
	s.glide.ReadFrame()

	for _, stg := range s.stages {
		stg.pitch.ReadFrame()
		stg.pulses.ReadFrame()
		stg.gateMode.ReadFrame()
		stg.glide.ReadFrame()
		stg.velocity.ReadFrame()
	}
}

func (s *stageSequence) postRead() {
	if outs := s.OutputsActive(); outs > 0 {
		s.reads = (s.reads + 1) % outs
	}
}

func (s *stageSequence) tick(times int, op func(int)) {
	clock := s.clock.LastFrame()
	reset := s.reset.LastFrame()
	mode := s.mode.LastFrame()

	for i := 0; i < times; i++ {
		if s.reads > 0 {
			op(i)
			continue
		}

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
		s.lastClock = clock[i]
		s.lastReset = reset[i]

		s.gateMode = mapGateMode(s.stages[s.stage].gateMode.LastFrame()[i])

		op(i)
	}
}

type stageSeqGate struct {
	*stageSequence
}

func (o *stageSeqGate) Read(out Frame) {
	o.read(out)
	clock := o.clock.LastFrame()
	o.tick(len(out), func(i int) {
		switch o.gateMode {
		case gateModeHold:
			out[i] = 1
		case gateModeRepeat:
			if clock[i] > 0 {
				out[i] = 1
			} else {
				out[i] = -1
			}
		case gateModeSingle:
			if o.pulse == 0 && clock[i] > 0 {
				out[i] = 1
			} else {
				out[i] = -1
			}
		case gateModeRest:
			out[i] = -1
		}
	})
	o.postRead()
}

type stageSeqPitch struct {
	*stageSequence
	slew *slew
}

func (o *stageSeqPitch) Read(out Frame) {
	o.read(out)
	o.tick(len(out), func(i int) {
		stage := o.stages[o.stage]
		in := stage.pitch.LastFrame()[i] * o.transpose.LastFrame()[i]
		glideAmount := o.glide.LastFrame()[i]
		if stageGlide := stage.glide.LastFrame(); stageGlide[i] > 0 {
			out[i] = o.slew.Tick(in, glideAmount, glideAmount)
		} else {
			out[i] = in
		}
	})
	o.postRead()
}

type stageSeqSync struct {
	*stageSequence
}

func (o *stageSeqSync) Read(out Frame) {
	o.read(out)
	o.tick(len(out), func(i int) {
		if o.stage == 0 && o.pulse == 0 {
			out[i] = 1
		} else {
			out[i] = -1
		}
	})
	o.postRead()
}

type stageSeqEndStage struct {
	*stageSequence
}

func (o *stageSeqEndStage) Read(out Frame) {
	o.read(out)
	o.tick(len(out), func(i int) {
		if o.lastStage != o.stage {
			out[i] = 1
		} else {
			out[i] = -1
		}
	})
	o.postRead()
}

type stageSeqVelocity struct {
	*stageSequence
	rolling Value
}

const averageVelocitySamples = 100

func (o *stageSeqVelocity) Read(out Frame) {
	o.read(out)
	o.tick(len(out), func(i int) {
		o.rolling -= o.rolling / averageVelocitySamples
		o.rolling += o.stages[o.stage].velocity.LastFrame()[i] / averageVelocitySamples
		out[i] = o.rolling
	})
	o.postRead()
}

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
