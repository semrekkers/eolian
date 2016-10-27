package module

func init() {
	Register("Clip", func(Config) (Patcher, error) { return NewClip() })
}

type Clip struct {
	IO
	in, max *In
}

func NewClip() (*Clip, error) {
	m := &Clip{
		in:  &In{Name: "input", Source: zero},
		max: &In{Name: "max", Source: NewBuffer(zero)},
	}
	err := m.Expose(
		[]*In{m.in, m.max},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Clip) Read(out Frame) {
	reader.in.Read(out)
	max := reader.max.ReadFrame()
	for i := range out {
		max := max[i]
		if out[i] > max {
			out[i] = max
		} else if out[i] < -max {
			out[i] = -max
		}
	}
}
