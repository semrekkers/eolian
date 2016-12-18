package module

func init() {
	Register("Direct", func(Config) (Patcher, error) { return NewDirect() })
}

type Direct struct {
	IO
	in *In
}

func NewDirect() (*Direct, error) {
	m := &Direct{
		in: &In{Name: "input", Source: zero},
	}
	err := m.Expose(
		[]*In{m.in},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Direct) Read(out Frame) {
	reader.in.Read(out)
}
