package module

func init() {
	Register("Invert", func(Config) (Patcher, error) { return newInvert() })
}

type invert struct {
	IO
	in *In
}

func newInvert() (*invert, error) {
	m := &invert{
		in: &In{Name: "input", Source: zero},
	}
	err := m.Expose(
		"Invert",
		[]*In{m.in},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (inv *invert) Read(out Frame) {
	inv.in.Read(out)
	for i := range out {
		out[i] = -out[i]
	}
}
