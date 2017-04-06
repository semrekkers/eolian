package module

import (
	"buddin.us/eolian/dsp"
	"github.com/mitchellh/mapstructure"
)

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
	avg               dsp.Float64
	smooth            bool
}

func newCtrl(min, max float64, smooth bool) (*ctrl, error) {
	m := &ctrl{
		in:     NewIn("input", dsp.Float64(0)),
		mod:    NewInBuffer("mod", dsp.Float64(1)),
		min:    NewInBuffer("min", dsp.Float64(min)),
		max:    NewInBuffer("max", dsp.Float64(max)),
		smooth: smooth,
	}
	err := m.Expose(
		"Control",
		[]*In{m.in, m.mod, m.min, m.max},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (c *ctrl) Process(out dsp.Frame) {
	c.in.Process(out)

	var (
		mod      = c.mod.ProcessFrame()
		min, max = c.min.ProcessFrame(), c.max.ProcessFrame()
	)

	for i := range out {
		m := dsp.Clamp(mod[i], -1, 1)

		if c.smooth {
			c.avg -= c.avg / averageVelocitySamples
			c.avg += out[i] / averageVelocitySamples
			out[i] = c.avg
		}
		out[i] = (out[i]*(max[i]-min[i]) + min[i]) * m
	}
}
