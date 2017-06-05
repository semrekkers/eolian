package module

import (
	"buddin.us/eolian/dsp"
)

func init() {
	Register("Decimate", func(Config) (Patcher, error) { return newDecimate() })
}

type decimate struct {
	IO
	in, rate, bits *In
	decimate       *dsp.Decimate
}

func newDecimate() (*decimate, error) {
	m := &decimate{
		in:       NewIn("input", dsp.Float64(0)),
		rate:     NewInBuffer("rate", dsp.Float64(44100)),
		bits:     NewInBuffer("bits", dsp.Float64(24)),
		decimate: &dsp.Decimate{},
	}

	err := m.Expose(
		"Decimate",
		[]*In{m.in, m.rate, m.bits},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (d *decimate) Process(out dsp.Frame) {
	d.in.Process(out)
	var (
		rate = d.rate.ProcessFrame()
		bits = d.bits.ProcessFrame()
	)

	for i := range out {
		out[i] = d.decimate.Tick(out[i], rate[i], bits[i])
	}
}
