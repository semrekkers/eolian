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
	IO
	clock, reset         *In
	steps                []*In
	size, reads          int
	step, lastStep       int
	lastClock, lastReset Value
}

func newGateSequence(steps int) (*gateSequence, error) {
	m := &gateSequence{
		clock:     &In{Name: "clock", Source: NewBuffer(zero)},
		reset:     &In{Name: "reset", Source: NewBuffer(zero)},
		size:      steps,
		steps:     make([]*In, steps),
		lastReset: -1,
		lastClock: -1,
		lastStep:  -1,
	}

	inputs := []*In{m.clock, m.reset}
	for i := 0; i < steps; i++ {
		m.steps[i] = &In{Name: fmt.Sprintf("%d/mode", i), Source: NewBuffer(zero)}
		inputs = append(inputs, m.steps[i])
	}

	outputs := []*Out{
		&Out{Name: "on", Provider: Provide(&gateSequencerOut{gateSequence: m, onBeat: true})},
		&Out{Name: "off", Provider: Provide(&gateSequencerOut{gateSequence: m, onBeat: false})},
	}

	return m, m.Expose("GateSequence", inputs, outputs)
}

func (s *gateSequence) read() {
	if s.reads > 0 {
		return
	}
	s.clock.ReadFrame()
	s.reset.ReadFrame()
	for _, s := range s.steps {
		s.ReadFrame()
	}
}

func (s *gateSequence) postRead() {
	if outs := s.OutputsActive(); outs > 0 {
		s.reads = (s.reads + 1) % outs
	}
}

func (s *gateSequence) tick(times int, op func(int)) {
	clock := s.clock.LastFrame()
	reset := s.reset.LastFrame()
	for i := 0; i < times; i++ {
		if s.reads > 0 {
			op(i)
			continue
		}

		if s.lastClock <= 0 && clock[i] > 0 {
			s.step = (s.step + 1) % s.size
		}
		if s.lastReset <= 0 && reset[i] > 0 {
			s.step = 0
		}
		s.lastClock = clock[i]
		s.lastReset = reset[i]

		op(i)

		s.lastStep = s.step
	}
}

type gateSequencerOut struct {
	*gateSequence
	onBeat bool
}

func (o *gateSequencerOut) Read(out Frame) {
	o.read()
	o.tick(len(out), func(i int) {
		mode := o.steps[o.step].LastFrame()[i]
		if o.step != o.lastStep {
			out[i] = -1
		} else {
			if o.onBeat && mode > 0 {
				out[i] = 1
			} else if !o.onBeat && mode <= 0 {
				out[i] = 1
			} else {
				out[i] = -1
			}
		}
	})
	o.postRead()
}
