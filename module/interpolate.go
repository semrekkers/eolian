package module

import "github.com/mitchellh/mapstructure"

func init() {
	Register("Interpolate", func(c Config) (Patcher, error) {
		var config interpolateConfig
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Max == 0 {
			config.Max = 1
		}
		return newInterpolate(config)
	})
}

type interpolateConfig struct {
	Min, Max Value
	Smooth   bool
}

type interpolate struct {
	IO
	in       *In
	min, max Value

	smooth  bool
	rolling Value
}

func newInterpolate(config interpolateConfig) (*interpolate, error) {
	m := &interpolate{
		in:     &In{Name: "input", Source: zero},
		max:    Value(config.Max),
		min:    Value(config.Min),
		smooth: config.Smooth,
	}
	err := m.Expose(
		"Interpolate",
		[]*In{m.in},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (m *interpolate) Read(out Frame) {
	m.in.Read(out)
	for i := range out {
		out[i] = out[i]*(m.max-m.min) + m.min
		if m.smooth {
			m.rolling -= m.rolling / averageVelocitySamples
			m.rolling += out[i] / averageVelocitySamples
			out[i] = m.rolling
		}
	}
}
