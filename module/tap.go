package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Tap", func(Config) (Patcher, error) { return newTap() })
}

type tap struct {
	IO
	in   *In
	tap  *Out
	side dsp.Frame
}

func newTap() (*tap, error) {
	m := &tap{
		in:   &In{Name: "input", Source: dsp.Float64(0), ForceSinking: true},
		side: dsp.NewFrame(),
	}
	m.tap = &Out{Name: "tap", Provider: dsp.Provide(&tapTap{m})}

	err := m.Expose(
		"Tap",
		[]*In{m.in},
		[]*Out{
			m.tap,
			{Name: "output", Provider: dsp.Provide(m)},
		},
	)
	return m, err
}

func (c *tap) Process(out dsp.Frame) {
	c.in.Process(out)
	for i := range out {
		c.side[i] = out[i]
	}
}

type tapTap struct {
	*tap
}

func (t *tapTap) Process(out dsp.Frame) {
	for i := range out {
		out[i] = t.side[i]
	}
}
