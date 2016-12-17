package module

func init() {
	Register("Clip", func(Config) (Patcher, error) { return NewClip() })
}

type Clip struct {
	IO
	in, level *In
}

func NewClip() (*Clip, error) {
	m := &Clip{
		in:    &In{Name: "input", Source: zero},
		level: &In{Name: "level", Source: NewBuffer(Value(1))},
	}
	err := m.Expose(
		[]*In{m.in, m.level},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Clip) Read(out Frame) {
	reader.in.Read(out)
	level := reader.level.ReadFrame()
	for i := range out {
		level := level[i]
		if out[i] > level {
			out[i] = level
		} else if out[i] < -level {
			out[i] = -level
		}
	}
}
