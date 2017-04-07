package module

import (
	"buddin.us/eolian/dsp"
	"github.com/mitchellh/mapstructure"
)

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
	lastTrigger, lastReset      dsp.Float64
}

func newCount(limit, step int) (*count, error) {
	m := &count{
		trigger:     NewInBuffer("trigger", dsp.Float64(-1)),
		reset:       NewInBuffer("reset", dsp.Float64(-1)),
		limit:       NewInBuffer("limit", dsp.Float64(limit)),
		step:        NewInBuffer("step", dsp.Float64(step)),
		lastTrigger: -1,
		lastReset:   -1,
	}
	return m, m.Expose(
		"Count",
		[]*In{m.trigger, m.reset, m.limit, m.step},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}})
}

func (c *count) Process(out dsp.Frame) {
	var (
		trigger = c.trigger.ProcessFrame()
		reset   = c.reset.ProcessFrame()
		limit   = c.limit.ProcessFrame()
		step    = c.step.ProcessFrame()
	)

	for i := range out {
		if c.lastReset < 0 && reset[i] > 0 {
			c.count = 0
		}
		if c.lastTrigger < 0 && trigger[i] > 0 {
			c.count = (c.count + int(step[i])) % int(limit[i])
		}
		out[i] = dsp.Float64(c.count)
		c.lastReset = reset[i]
		c.lastTrigger = trigger[i]
	}
}
