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

	smooth bool
	after  [2]Value
}

func NewInterpolate(config InterpolateConfig) (*Interpolate, error) {
	m := &Interpolate{
		in:     &In{Name: "input", Source: zero},
		max:    &In{Name: "max", Source: NewBuffer(config.Max)},
		min:    &In{Name: "min", Source: NewBuffer(config.Min)},
		smooth: config.Smooth,
		after:  [2]Value{},
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
		if reader.smooth {
			reader.after[0] += (-reader.after[0] + out[i]) * 0.5
			reader.after[1] += (-reader.after[1] + reader.after[0]) * 0.5
			out[i] = reader.after[1]
		}
	}
}
