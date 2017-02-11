package module

import "github.com/mitchellh/mapstructure"

func init() {
	Register("Control", func(c Config) (Patcher, error) {
		var config struct {
			Min, Max float64
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		return newCtrl(config.Min, config.Max)
	})
}

type ctrl struct {
	IO
	ctrl, mod         *In
	min, max, ctrlAvg Value
}

func newCtrl(min, max float64) (*ctrl, error) {
	m := &ctrl{
		ctrl: &In{Name: "control", Source: NewBuffer(zero)},
		mod:  &In{Name: "mod", Source: NewBuffer(Value(1))},
		min:  Value(min),
		max:  Value(max),
	}
	err := m.Expose(
		"ctrl",
		[]*In{m.ctrl, m.mod},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (c *ctrl) Read(out Frame) {
	var (
		ctrl, mod      = c.ctrl.ReadFrame(), c.mod.ReadFrame()
		_, unmodulated = c.mod.Source.(*Buffer).Reader.(Valuer)
		in             Value
	)

	for i := range out {
		c.ctrlAvg -= c.ctrlAvg / averageVelocitySamples
		c.ctrlAvg += ctrl[i] / averageVelocitySamples
		if unmodulated {
			in = c.ctrlAvg
		} else {
			in = mod[i] * c.ctrlAvg
		}
		if c.max == 0 && c.min == 0 {
			out[i] = in
		} else {
			out[i] = in*(c.max-c.min) + c.min
		}
	}
}
