package module

import "github.com/mitchellh/mapstructure"

func init() {
	Register("Loop", func(c Config) (Patcher, error) {
		var config struct {
			Max int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Max == 0 {
			config.Max = 10
		}
		return NewLoop(config.Max)
	})
}

type Loop struct {
	IO
	in, trigger, reset *In

	max, stop              int
	memory                 []Value
	offset                 int
	started, recording     bool
	lastTrigger, lastReset Value
}

func NewLoop(max int) (*Loop, error) {
	length := max * SampleRate

	m := &Loop{
		in:          &In{Name: "input", Source: zero},
		trigger:     &In{Name: "trigger", Source: NewBuffer(zero)},
		reset:       &In{Name: "reset", Source: NewBuffer(zero)},
		memory:      make([]Value, length),
		stop:        length,
		max:         length,
		lastTrigger: -1,
		lastReset:   -1,
	}
	err := m.Expose(
		[]*In{m.in, m.trigger, m.reset},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Loop) Read(out Frame) {
	reader.in.Read(out)
	trigger := reader.trigger.ReadFrame()
	reset := reader.reset.ReadFrame()
	for i := range out {
		if reader.lastReset < 0 && reset[i] > 0 {
			reader.started = false
			for i := range reader.memory {
				reader.memory[i] = 0
				reader.offset = 0
			}
		}

		if reader.lastTrigger < 0 && trigger[i] > 0 {
			reader.started = true
			reader.recording = !reader.recording
			if !reader.recording {
				reader.stop = reader.offset
				reader.offset = 0
			}
		}
		if !reader.started {
			continue
		}

		if reader.recording {
			reader.memory[reader.offset] += out[i]
			reader.offset = (reader.offset + 1) % len(reader.memory)
		} else {
			out[i] = reader.memory[reader.offset] + out[i]
			reader.offset = (reader.offset + 1) % reader.stop
		}
		reader.lastTrigger = trigger[i]
	}
}
