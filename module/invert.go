package module

func init() {
	Register("Invert", func(Config) (Patcher, error) { return NewInvert() })
}

type Invert struct {
	IO
	in *In
}

func NewInvert() (*Invert, error) {
	m := &Invert{
		in: &In{Name: "input", Source: zero},
	}
	err := m.Expose(
		[]*In{m.in},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (inv *Invert) Read(out Frame) {
	inv.in.Read(out)
	for i := range out {
		out[i] = -out[i]
	}
}
