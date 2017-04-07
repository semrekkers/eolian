package module

import (
	"fmt"

	"buddin.us/eolian/dsp"

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
	lastClock, lastReset dsp.Float64

	onBeatOut, offBeatOut dsp.Frame
}

func newGateSequence(steps int) (*gateSequence, error) {
	m := &gateSequence{
		clock:      NewInBuffer("clock", dsp.Float64(0)),
		reset:      NewInBuffer("reset", dsp.Float64(0)),
		size:       steps,
		steps:      make([]*In, steps),
		onBeatOut:  dsp.NewFrame(),
		offBeatOut: dsp.NewFrame(),
		lastReset:  -1,
		lastClock:  -1,
		lastStep:   -1,
	}

	inputs := []*In{m.clock, m.reset}
	for i := 0; i < steps; i++ {
		m.steps[i] = NewInBuffer(fmt.Sprintf("%d/mode", i), dsp.Float64(0))
		inputs = append(inputs, m.steps[i])
	}

	outputs := []*Out{
		&Out{Name: "on", Provider: provideCopyOut(m, &m.onBeatOut)},
		&Out{Name: "off", Provider: provideCopyOut(m, &m.offBeatOut)},
	}

	return m, m.Expose("GateSequence", inputs, outputs)
}

func (s *gateSequence) Process(out dsp.Frame) {
	s.incrRead(func() {
		clock := s.clock.ProcessFrame()
		reset := s.reset.ProcessFrame()
		for _, s := range s.steps {
			s.ProcessFrame()
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
