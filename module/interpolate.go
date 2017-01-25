package module

import "github.com/mitchellh/mapstructure"

func init() {
	Register("Interpolate", func(c Config) (Patcher, error) {
		var config InterpolateConfig
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Max == 0 {
			config.Max = 1
		}
		return NewInterpolate(config)
	})
}

type InterpolateConfig struct {
	Min, Max Value
	Smooth   bool
}

type Interpolate struct {
	IO
	in, max, min *In

	smooth  bool
	rolling Value
}

func NewInterpolate(config InterpolateConfig) (*Interpolate, error) {
	m := &Interpolate{
		in:     &In{Name: "input", Source: zero},
		max:    &In{Name: "max", Source: NewBuffer(config.Max)},
		min:    &In{Name: "min", Source: NewBuffer(config.Min)},
		smooth: config.Smooth,
	}
	err := m.Expose(
		"Interpolate",
		[]*In{m.in, m.max, m.min},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (interp *Interpolate) Read(out Frame) {
	interp.in.Read(out)
	max := interp.max.ReadFrame()
	min := interp.min.ReadFrame()

	for i := range out {
		out[i] = out[i]*(max[i]-min[i]) + min[i]
		if interp.smooth {
			interp.rolling -= interp.rolling / averageVelocitySamples
			interp.rolling += out[i] / averageVelocitySamples
			out[i] = interp.rolling
		}
	}
}
