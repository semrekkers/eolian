package module

func init() {
	Register("Direct", func(Config) (Patcher, error) { return NewDirect() })
}

type Direct struct {
	IO
	In *In
}

func NewDirect() (*Direct, error) {
	m := &Direct{
		In: &In{Name: "input", Source: zero},
	}
	err := m.Expose(
		[]*In{m.In},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Direct) Read(out Frame) {
	reader.In.Read(out)
}
