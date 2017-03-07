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
	ctrl, mod     *In
	min, max, avg Value
	smooth        bool
}

func newCtrl(min, max float64, smooth bool) (*ctrl, error) {
	m := &ctrl{
		ctrl:   &In{Name: "control", Source: NewBuffer(zero)},
		mod:    &In{Name: "mod", Source: NewBuffer(Value(1))},
		min:    Value(min),
		max:    Value(max),
		smooth: smooth,
	}
	err := m.Expose(
		"Control",
		[]*In{m.ctrl, m.mod},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (c *ctrl) Read(out Frame) {
	var (
		ctrl, mod = c.ctrl.ReadFrame(), c.mod.ReadFrame()
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
		if c.max == 0 && c.min == 0 {
			out[i] = in
		} else {
			out[i] = in*(c.max-c.min) + c.min
		}
	}
}
