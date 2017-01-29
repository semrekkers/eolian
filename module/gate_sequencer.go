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
	clock, reset *In
	steps        []*In

	size, step, reads    int
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
	}

	inputs := []*In{m.clock, m.reset}
	for i := 0; i < steps; i++ {
		m.steps[i] = &In{Name: fmt.Sprintf("%d/mode", i), Source: NewBuffer(zero)}
		inputs = append(inputs, m.steps[i])
	}

	outputs := []*Out{
		&Out{Name: "on", Provider: Provide(
			&gateSequencerOut{
				gateSequence: m,
				onBeat:       true,
				lastStep:     -1,
			})},
		&Out{Name: "off", Provider: Provide(
			&gateSequencerOut{
				gateSequence: m,
				onBeat:       false,
				lastStep:     -1,
			})},
	}

	return m, m.Expose("GateSequence", inputs, outputs)
}

func (s *gateSequence) read(out Frame) {
	if s.reads == 0 {
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
			s.lastClock = clock[i]
			s.lastReset = reset[i]
		}
	}
	if outs := s.OutputsActive(); outs > 0 {
		s.reads = (s.reads + 1) % outs
	}
}

type gateSequencerOut struct {
	*gateSequence
	onBeat   bool
	lastStep int
}

func (o *gateSequencerOut) Read(out Frame) {
	o.read(out)
	for i := range out {
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
		o.lastStep = o.step
	}
}
