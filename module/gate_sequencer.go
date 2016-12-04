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
		return NewGateSequencer(config.Steps)
	})
}

type GateSequencer struct {
	IO
	clock, reset *In
	steps        []*In

	size, step, reads    int
	lastClock, lastReset Value
}

func NewGateSequencer(steps int) (*GateSequencer, error) {
	m := &GateSequencer{
		clock:     &In{Name: "clock", Source: NewBuffer(zero)},
		reset:     &In{Name: "reset", Source: NewBuffer(zero)},
		size:      steps,
		steps:     make([]*In, steps),
		lastReset: -1,
		lastClock: -1,
	}

	inputs := []*In{m.clock, m.reset}
	for i := 0; i < steps; i++ {
		m.steps[i] = &In{Name: fmt.Sprintf("%d.status", i), Source: NewBuffer(zero)}
		inputs = append(inputs, m.steps[i])
	}

	if err := m.Expose(inputs, []*Out{
		&Out{Name: "on", Provider: Provide(
			&gateSequencerOut{
				GateSequencer: m,
				onBeat:        true,
				lastStep:      -1,
			})},
		&Out{Name: "off", Provider: Provide(
			&gateSequencerOut{
				GateSequencer: m,
				onBeat:        false,
				lastStep:      -1,
			})},
	}); err != nil {
		return nil, err
	}

	return m, nil
}

func (s *GateSequencer) read(out Frame) {
	if s.reads == 0 {
		clock := s.clock.ReadFrame()
		reset := s.reset.ReadFrame()
		for _, s := range s.steps {
			s.ReadFrame()
		}
		for i := range out {
			if s.lastClock < 0 && clock[i] > 0 {
				s.step = (s.step + 1) % s.size
			}
			if s.lastReset < 0 && reset[i] > 0 {
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
	*GateSequencer
	onBeat   bool
	lastStep int
}

func (reader *gateSequencerOut) Read(out Frame) {
	reader.read(out)
	for i := range out {
		status := reader.steps[reader.step].LastFrame()[i]
		if reader.step != reader.lastStep {
			out[i] = -1
		} else {
			if reader.onBeat && status > 0 {
				out[i] = 1
			} else if !reader.onBeat && status <= 0 {
				out[i] = 1
			} else {
				out[i] = -1
			}
		}
		reader.lastStep = reader.step
	}
}
