package module

func init() {
	Register("Clip", func(Config) (Patcher, error) { return newClip() })
}

type clip struct {
	IO
	in, level *In
}

func newClip() (*clip, error) {
	m := &clip{
		in:    &In{Name: "input", Source: zero},
		level: &In{Name: "level", Source: NewBuffer(Value(1))},
	}
	err := m.Expose(
		"Clip",
		[]*In{m.in, m.level},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (c *clip) Read(out Frame) {
	c.in.Read(out)
	level := c.level.ReadFrame()
	for i := range out {
		level := level[i]
		if out[i] > level {
			out[i] = level
		} else if out[i] < -level {
			out[i] = -level
		}
	}
}
