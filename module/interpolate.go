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
}

type Interpolate struct {
	IO
	in, max, min *In
}

func NewInterpolate(config InterpolateConfig) (*Interpolate, error) {
	m := &Interpolate{
		in:  &In{Name: "input", Source: zero},
		max: &In{Name: "max", Source: NewBuffer(config.Max)},
		min: &In{Name: "min", Source: NewBuffer(config.Min)},
	}
	err := m.Expose(
		[]*In{m.in, m.max, m.min},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Interpolate) Read(out Frame) {
	reader.in.Read(out)
	max := reader.max.ReadFrame()
	min := reader.min.ReadFrame()

	for i := range out {
		out[i] = out[i]*(max[i]-min[i]) + min[i]
	}
}
