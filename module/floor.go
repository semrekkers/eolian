package module

func init() {
	Register("Floor", func(Config) (Patcher, error) { return newFloor() })
}

type floor struct {
	IO
	in, level *In
}

func newFloor() (*floor, error) {
	m := &floor{
		in: &In{Name: "input", Source: zero},
	}
	err := m.Expose(
		"Floor",
		[]*In{m.in},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (w *floor) Read(out Frame) {
	w.in.Read(out)
	for i := range out {
		out[i] = floorValue(out[i])
	}
}
