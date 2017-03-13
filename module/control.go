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
		return newCtrl(config.Min, config.Max, config.Smooth)
	})
}

type ctrl struct {
	IO
	ctrl, mod, min, max *In
	avg                 Value
	smooth              bool
}

func newCtrl(min, max float64, smooth bool) (*ctrl, error) {
	m := &ctrl{
		ctrl:   &In{Name: "control", Source: NewBuffer(zero)},
		mod:    &In{Name: "mod", Source: NewBuffer(Value(1))},
		min:    &In{Name: "min", Source: NewBuffer(Value(0))},
		max:    &In{Name: "max", Source: NewBuffer(Value(1))},
		smooth: smooth,
	}
	err := m.Expose(
		"Control",
		[]*In{m.ctrl, m.mod, m.min, m.max},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (c *ctrl) Read(out Frame) {
	var (
		ctrl, mod = c.ctrl.ReadFrame(), c.mod.ReadFrame()
		min, max  = c.min.ReadFrame(), c.max.ReadFrame()
		_, static = c.mod.Source.(*Buffer).Reader.(Valuer)
		in        Value
	)

	for i := range out {
		if c.smooth {
			c.avg -= c.avg / averageVelocitySamples
			c.avg += ctrl[i] / averageVelocitySamples
			in = c.avg
		} else {
			in = ctrl[i]
		}
		if !static {
			in *= mod[i]
		}
		out[i] = in*(max[i]-min[i]) + min[i]
	}
}
