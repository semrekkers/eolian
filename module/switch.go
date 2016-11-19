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
	trigger, reset *In
	sources        []*In

	step                   int
	lastTrigger, lastReset Value
}

func NewSwitch(size int) (*Switch, error) {
	m := &Switch{
		trigger:     &In{Name: "trigger", Source: NewBuffer(zero)},
		reset:       &In{Name: "reset", Source: NewBuffer(zero)},
		lastTrigger: -1,
	}
	inputs := []*In{m.trigger, m.reset}
	for i := 0; i < size; i++ {
		in := &In{
			Name:   fmt.Sprintf("%d.input", i),
			Source: NewBuffer(zero),
		}
		m.sources = append(m.sources, in)
		inputs = append(inputs, in)
	}
	err := m.Expose(inputs, []*Out{{Name: "output", Provider: Provide(m)}})
	return m, err
}

func (reader *Switch) Read(out Frame) {
	trigger := reader.trigger.ReadFrame()
	reset := reader.trigger.ReadFrame()
	for i := 0; i < len(reader.sources); i++ {
		reader.sources[i].ReadFrame()
	}
	for i := range out {
		if reader.lastReset < 0 && reset[i] > 0 {
			reader.step = 0
		} else if reader.lastTrigger < 0 && trigger[i] > 0 {
			reader.step = (reader.step + 1) % len(reader.sources)
		}
		out[i] = reader.sources[reader.step].LastFrame()[i]
		reader.lastTrigger = trigger[i]
	}
}
