package module

import (
	"buddin.us/eolian/dsp"
	"github.com/mitchellh/mapstructure"
)

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
	Min, Max dsp.Float64
	Smooth   bool
}

type interpolate struct {
	IO
	in       *In
	min, max dsp.Float64

	smooth  bool
	rolling dsp.Float64
}

func newInterpolate(config interpolateConfig) (*interpolate, error) {
	m := &interpolate{
		in:     NewIn("input", dsp.Float64(0)),
		max:    dsp.Float64(config.Max),
		min:    dsp.Float64(config.Min),
		smooth: config.Smooth,
	}
	err := m.Expose(
		"Interpolate",
		[]*In{m.in},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (m *interpolate) Process(out dsp.Frame) {
	m.in.Process(out)
	for i := range out {
		out[i] = out[i]*(m.max-m.min) + m.min
		if m.smooth {
			m.rolling -= m.rolling / averageVelocitySamples
			m.rolling += out[i] / averageVelocitySamples
			out[i] = m.rolling
		}
	}
}
