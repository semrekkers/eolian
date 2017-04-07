package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Crossfade", func(Config) (Patcher, error) { return newCrossfade() })
}

type crossfade struct {
	IO
	a, b, bias *In
}

func newCrossfade() (*crossfade, error) {
	m := &crossfade{
		a:    NewInBuffer("a", dsp.Float64(0)),
		b:    NewInBuffer("b", dsp.Float64(0)),
		bias: NewInBuffer("bias", dsp.Float64(0)),
	}
	err := m.Expose(
		"Crossfade",
		[]*In{m.a, m.b, m.bias},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (c *crossfade) Process(out dsp.Frame) {
	a, b := c.a.ProcessFrame(), c.b.ProcessFrame()
	bias := c.bias.ProcessFrame()
	for i := range out {
		out[i] = dsp.CrossSum(bias[i], a[i], b[i])
	}
}
