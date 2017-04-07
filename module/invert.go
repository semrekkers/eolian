package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Invert", func(Config) (Patcher, error) { return newInvert() })
}

type invert struct {
	IO
	in *In
}

func newInvert() (*invert, error) {
	m := &invert{
		in: NewIn("input", dsp.Float64(0)),
	}
	err := m.Expose(
		"Invert",
		[]*In{m.in},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (inv *invert) Process(out dsp.Frame) {
	inv.in.Process(out)
	for i := range out {
		out[i] = -out[i]
	}
}
