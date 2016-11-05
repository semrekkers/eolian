package module

func init() {
	Register("Interpolate", func(Config) (Patcher, error) { return NewInterpolate() })
}

type Interpolate struct {
	IO
	in, max, min *In
}

func NewInterpolate() (*Interpolate, error) {
	m := &Interpolate{
		in:  &In{Name: "input", Source: zero},
		max: &In{Name: "max", Source: NewBuffer(Value(1))},
		min: &In{Name: "min", Source: NewBuffer(zero)},
	}
	err := m.Expose(
		[]*In{m.in, m.max, m.min},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Interpolate) Read(out Frame) {
	reader.in.Read(out)
	max := reader.max.ReadFrame()
	min := reader.min.ReadFrame()

	for i := range out {
		out[i] = out[i]*(max[i]-min[i]) + min[i]
	}
}
