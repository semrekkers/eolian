package module

func init() {
	Register("Tap", func(Config) (Patcher, error) { return newTap() })
}

type tap struct {
	IO
	in   *In
	tap  *Out
	side Frame
}

func newTap() (*tap, error) {
	m := &tap{
		in:   &In{Name: "input", Source: zero},
		side: make(Frame, FrameSize),
	}
	m.tap = &Out{Name: "tap", Provider: Provide(&tapTap{m})}

	err := m.Expose(
		"Tap",
		[]*In{m.in},
		[]*Out{
			m.tap,
			{Name: "output", Provider: Provide(m)},
		},
	)
	return m, err
}

func (c *tap) Read(out Frame) {
	c.in.Read(out)
	for i := range out {
		c.side[i] = out[i]
	}
}

type tapTap struct {
	*tap
}

func (t *tapTap) Read(out Frame) {
	for i := range out {
		out[i] = t.side[i]
	}
}
