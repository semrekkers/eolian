package module

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Switch", func(c Config) (Patcher, error) {
		var config struct {
			Size int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 4
		}
		return NewSwitch(config.Size)
	})
}

type Switch struct {
	IO
	clock, reset *In
	sources      []*In

	step                 int
	lastClock, lastReset Value
}

func NewSwitch(size int) (*Switch, error) {
	m := &Switch{
		clock:     &In{Name: "clock", Source: NewBuffer(zero)},
		reset:     &In{Name: "reset", Source: NewBuffer(zero)},
		lastClock: -1,
	}
	inputs := []*In{m.clock, m.reset}
	for i := 0; i < size; i++ {
		in := &In{
			Name:   fmt.Sprintf("%d/input", i),
			Source: NewBuffer(zero),
		}
		m.sources = append(m.sources, in)
		inputs = append(inputs, in)
	}
	err := m.Expose(inputs, []*Out{{Name: "output", Provider: Provide(m)}})
	return m, err
}

func (s *Switch) Read(out Frame) {
	clock := s.clock.ReadFrame()
	reset := s.reset.ReadFrame()
	for i := 0; i < len(s.sources); i++ {
		s.sources[i].ReadFrame()
	}
	for i := range out {
		if s.lastReset < 0 && reset[i] > 0 {
			s.step = 0
		} else if s.lastClock < 0 && clock[i] > 0 {
			s.step = (s.step + 1) % len(s.sources)
		}
		out[i] = s.sources[s.step].LastFrame()[i]
		s.lastClock = clock[i]
	}
}
