package module

import "github.com/mitchellh/mapstructure"

func init() {
	Register("Control", func(c Config) (Patcher, error) {
		var config struct {
			Min, Max float64
			Smooth   bool
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Max == 0 {
			config.Max = 1
		}
		return newCtrl(config.Min, config.Max, config.Smooth)
	})
}

type ctrl struct {
	IO
	in, mod, min, max *In
	avg               Value
	smooth            bool
}

func newCtrl(min, max float64, smooth bool) (*ctrl, error) {
	m := &ctrl{
		in:     &In{Name: "input", Source: zero},
		mod:    &In{Name: "mod", Source: NewBuffer(Value(1))},
		min:    &In{Name: "min", Source: NewBuffer(Value(min))},
		max:    &In{Name: "max", Source: NewBuffer(Value(max))},
		smooth: smooth,
	}
	err := m.Expose(
		"Control",
		[]*In{m.in, m.mod, m.min, m.max},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (c *ctrl) Read(out Frame) {
	c.in.Read(out)

	var (
		mod      = c.mod.ReadFrame()
		min, max = c.min.ReadFrame(), c.max.ReadFrame()
	)

	for i := range out {
		m := clampValue(mod[i], -1, 1)

		if c.smooth {
			c.avg -= c.avg / averageVelocitySamples
			c.avg += out[i] / averageVelocitySamples
			out[i] = c.avg
		}
		out[i] = (out[i]*(max[i]-min[i]) + min[i]) * m
	}
}
