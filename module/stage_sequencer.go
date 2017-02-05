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

	reads    int
	outState []*stageSeqOut
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
		outState:  make([]*stageSeqOut, 5),
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

	for i := 0; i < len(m.outState); i++ {
		m.outState[i] = &stageSeqOut{
			lastClock: -1,
			lastReset: -1,
			lastStage: -1,
			pulse:     -1,
		}
	}

	return m, m.Expose(
		"StageSequence",
		inputs,
		[]*Out{
			{Name: "gate", Provider: Provide(&stageSeqGate{stageSequence: m, index: 0})},
			{Name: "pitch", Provider: Provide(&stageSeqPitch{stageSequence: m, index: 1, slew: newSlew()})},
			{Name: "velocity", Provider: Provide(&stageSeqVelocity{stageSequence: m, index: 2})},
			{Name: "endstage", Provider: Provide(&stageSeqEndStage{stageSequence: m, index: 3})},
			{Name: "sync", Provider: Provide(&stageSeqSync{stageSequence: m, index: 4})},
		},
	)
}

func (s *stageSequence) read(out Frame) {
	if s.reads == 0 {
		s.clock.ReadFrame()
		s.reset.ReadFrame()
		s.mode.ReadFrame()
		s.transpose.ReadFrame()
		s.glide.ReadFrame()

		for i := 0; i < len(s.stages); i++ {
			s.stages[i].pitch.ReadFrame()
			s.stages[i].pulses.ReadFrame()
			s.stages[i].gateMode.ReadFrame()
			s.stages[i].glide.ReadFrame()
			s.stages[i].velocity.ReadFrame()
		}
	}
	if outs := s.OutputsActive(); outs > 0 {
		s.reads = (s.reads + 1) % outs
	}
}

type stageSeqOut struct {
	stage, lastStage     int
	pulse, gateMode      int
	lastClock, lastReset Value
	pong                 bool
}

func (s *stageSeqOut) advance(seq *stageSequence, i int) {
	clock := seq.clock.LastFrame()
	reset := seq.reset.LastFrame()
	mode := seq.mode.LastFrame()

	if s.lastClock < 0 && clock[i] > 0 {
		pulses := seq.stages[s.stage].pulses.LastFrame()[i]
		lastPulse := s.pulse
		s.pulse = (s.pulse + 1) % int(pulses)

		if lastPulse >= 0 && s.pulse == 0 {
			s.lastStage = s.stage
			switch mapPatternMode(mode[i]) {
			case patternModeSequential:
				s.stage = (s.stage + 1) % len(seq.stages)
				s.pong = false
			case patternModePingPong:
				var inc = 1
				if s.pong {
					inc = -1
				}
				s.stage += inc

				if s.stage == len(seq.stages)-1 {
					s.stage = len(seq.stages) - 1
					s.pong = true
				} else if s.stage == 0 {
					s.stage = 0
					s.pong = false
				}
			case patternModeRandom:
				s.stage = rand.Intn(len(seq.stages))
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

	s.gateMode = mapGateMode(seq.stages[s.stage].gateMode.LastFrame()[i])
}

type stageSeqGate struct {
	*stageSequence
	index int
}

func (o *stageSeqGate) Read(out Frame) {
	o.read(out)
	clock := o.clock.LastFrame()
	state := o.outState[o.index]

	for i := range out {
		state.advance(o.stageSequence, i)

		switch state.gateMode {
		case gateModeHold:
			out[i] = 1
		case gateModeRepeat:
			if clock[i] > 0 {
				out[i] = 1
			} else {
				out[i] = -1
			}
		case gateModeSingle:
			if state.pulse == 0 && clock[i] > 0 {
				out[i] = 1
			} else {
				out[i] = -1
			}
		case gateModeRest:
			out[i] = -1
		}
	}
}

type stageSeqPitch struct {
	*stageSequence
	slew  *slew
	index int
}

func (o *stageSeqPitch) Read(out Frame) {
	o.read(out)
	state := o.outState[o.index]
	for i := range out {
		state.advance(o.stageSequence, i)
		stage := o.stages[state.stage]
		in := stage.pitch.LastFrame()[i] * o.transpose.LastFrame()[i]
		var amt Value
		if glide := stage.glide.LastFrame(); glide[i] > 0 {
			amt = glide[i]
		}
		out[i] = o.slew.Tick(in, amt, amt)
	}
}

type stageSeqSync struct {
	*stageSequence
	index int
}

func (o *stageSeqSync) Read(out Frame) {
	o.read(out)
	state := o.outState[o.index]
	for i := range out {
		state.advance(o.stageSequence, i)
		if state.stage == 0 && state.pulse == 0 {
			out[i] = 1
		} else {
			out[i] = -1
		}
	}
}

type stageSeqEndStage struct {
	*stageSequence
	index int
}

func (o *stageSeqEndStage) Read(out Frame) {
	o.read(out)
	state := o.outState[o.index]
	for i := range out {
		state.advance(o.stageSequence, i)
		if state.lastStage != state.stage {
			out[i] = 1
		} else {
			out[i] = -1
		}
	}
}

type stageSeqVelocity struct {
	*stageSequence
	index   int
	rolling Value
}

const averageVelocitySamples = 100

func (o *stageSeqVelocity) Read(out Frame) {
	o.read(out)
	state := o.outState[o.index]
	for i := range out {
		state.advance(o.stageSequence, i)
		o.rolling -= o.rolling / averageVelocitySamples
		o.rolling += o.stages[state.stage].velocity.LastFrame()[i] / averageVelocitySamples
		out[i] = o.rolling
	}
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
