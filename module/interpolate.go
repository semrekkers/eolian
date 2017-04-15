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
	average  dsp.RollingAverage
	smooth   bool
}

func newInterpolate(config interpolateConfig) (*interpolate, error) {
	m := &interpolate{
		in:      NewIn("input", dsp.Float64(0)),
		max:     dsp.Float64(config.Max),
		min:     dsp.Float64(config.Min),
		smooth:  config.Smooth,
		average: dsp.RollingAverage{Window: averageVelocitySamples},
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
		out[i] = dsp.Lerp(out[i], m.min, m.max)
		if m.smooth {
			out[i] = m.average.Tick(out[i])
		}
	}
}
