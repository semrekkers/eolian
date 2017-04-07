package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Direct", func(Config) (Patcher, error) { return newDirect() })
}

type direct struct {
	IO
	in *In
}

func newDirect() (*direct, error) {
	m := &direct{
		in: NewIn("input", dsp.Float64(0)),
	}
	err := m.Expose(
		"Direct",
		[]*In{m.in},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (d *direct) Process(out dsp.Frame) {
	d.in.Process(out)
}
