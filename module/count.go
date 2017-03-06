package module

import "github.com/mitchellh/mapstructure"

func init() {
	Register("Count", func(c Config) (Patcher, error) {
		var config struct {
			Limit int
			Step  int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Limit == 0 {
			config.Limit = 1024
		}
		if config.Step == 0 {
			config.Step = 1
		}
		return newCount(config.Limit, config.Step)
	})
}

type count struct {
	IO
	count                       int
	trigger, reset, limit, step *In
	lastTrigger, lastReset      Value
}

func newCount(limit, step int) (*count, error) {
	m := &count{
		trigger:     &In{Name: "trigger", Source: NewBuffer(Value(-1))},
		reset:       &In{Name: "reset", Source: NewBuffer(Value(-1))},
		limit:       &In{Name: "limit", Source: NewBuffer(Value(limit))},
		step:        &In{Name: "step", Source: NewBuffer(Value(step))},
		lastTrigger: -1,
		lastReset:   -1,
	}
	return m, m.Expose(
		"Count",
		[]*In{m.trigger, m.reset, m.limit, m.step},
		[]*Out{{Name: "output", Provider: Provide(m)}})
}

func (c *count) Read(out Frame) {
	var (
		trigger = c.trigger.ReadFrame()
		reset   = c.reset.ReadFrame()
		limit   = c.limit.ReadFrame()
		step    = c.step.ReadFrame()
	)

	for i := range out {
		if c.lastReset < 0 && reset[i] > 0 {
			c.count = 0
		}
		if c.lastTrigger < 0 && trigger[i] > 0 {
			c.count = (c.count + int(step[i])) % int(limit[i])
		}
		out[i] = Value(c.count)
		c.lastReset = reset[i]
		c.lastTrigger = trigger[i]
	}
}
