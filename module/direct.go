package module

func init() {
	Register("Direct", func(Config) (Patcher, error) { return newDirect() })
}

type direct struct {
	IO
	in *In
}

func newDirect() (*direct, error) {
	m := &direct{
		in: &In{Name: "input", Source: zero},
	}
	err := m.Expose(
		"Direct",
		[]*In{m.in},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (d *direct) Read(out Frame) {
	d.in.Read(out)
}
