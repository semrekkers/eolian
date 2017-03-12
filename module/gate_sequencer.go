package module

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("GateSequence", func(c Config) (Patcher, error) {
		var config struct {
			Steps int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Steps == 0 {
			config.Steps = 16
		}
		return newGateSequence(config.Steps)
	})
}

type gateSequence struct {
	multiOutIO
	clock, reset         *In
	steps                []*In
	size                 int
	step, lastStep       int
	lastClock, lastReset Value

	onBeatOut, offBeatOut Frame
}

func newGateSequence(steps int) (*gateSequence, error) {
	m := &gateSequence{
		clock:      &In{Name: "clock", Source: NewBuffer(zero)},
		reset:      &In{Name: "reset", Source: NewBuffer(zero)},
		size:       steps,
		steps:      make([]*In, steps),
		onBeatOut:  make(Frame, FrameSize),
		offBeatOut: make(Frame, FrameSize),
		lastReset:  -1,
		lastClock:  -1,
		lastStep:   -1,
	}

	inputs := []*In{m.clock, m.reset}
	for i := 0; i < steps; i++ {
		m.steps[i] = &In{Name: fmt.Sprintf("%d/mode", i), Source: NewBuffer(zero)}
		inputs = append(inputs, m.steps[i])
	}

	outputs := []*Out{
		&Out{Name: "on", Provider: provideCopyOut(m, &m.onBeatOut)},
		&Out{Name: "off", Provider: provideCopyOut(m, &m.offBeatOut)},
	}

	return m, m.Expose("GateSequence", inputs, outputs)
}

func (s *gateSequence) Read(out Frame) {
	s.incrRead(func() {
		clock := s.clock.ReadFrame()
		reset := s.reset.ReadFrame()
		for _, s := range s.steps {
			s.ReadFrame()
		}
		for i := range out {
			if s.lastClock <= 0 && clock[i] > 0 {
				s.step = (s.step + 1) % s.size
			}
			if s.lastReset <= 0 && reset[i] > 0 {
				s.step = 0
			}

			mode := s.steps[s.step].LastFrame()[i]
			if s.step != s.lastStep {
				s.onBeatOut[i] = -1
				s.offBeatOut[i] = -1
			} else {
				if mode > 0 {
					s.onBeatOut[i] = 1
					s.offBeatOut[i] = -1
				} else if mode <= 0 {
					s.onBeatOut[i] = -1
					s.offBeatOut[i] = 1
				} else {
					s.onBeatOut[i] = -1
					s.offBeatOut[i] = -1
				}
			}

			s.lastClock = clock[i]
			s.lastReset = reset[i]
			s.lastStep = s.step
		}
	})
}
