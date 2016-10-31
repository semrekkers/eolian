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

func (reader *Invert) Read(out Frame) {
	reader.in.Read(out)
	for i := range out {
		out[i] = -out[i]
	}
}
