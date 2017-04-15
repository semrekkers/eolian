package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Tap", func(Config) (Patcher, error) { return newTap() })
}

type tap struct {
	IO
	in, tap *In
	tapOut  *Out
	side    dsp.Frame
}

func newTap() (*tap, error) {
	m := &tap{
		in:   &In{Name: "input", Source: dsp.Float64(0)},
		tap:  &In{Name: "tap", Source: dsp.NewBuffer(dsp.Float64(0)), ForceSinking: true},
		side: dsp.NewFrame(),
	}
	m.tapOut = &Out{Name: "tap", Provider: dsp.Provide(&tapTap{m})}

	err := m.Expose(
		"Tap",
		[]*In{m.in, m.tap},
		[]*Out{
			m.tapOut,
			{Name: "output", Provider: dsp.Provide(m)},
		},
	)
	return m, err
}

func (c *tap) Process(out dsp.Frame) {
	c.in.Process(out)
	tap := c.tap.ProcessFrame()
	for i := range out {
		if isNormal(c.tap) {
			c.side[i] = out[i]
		} else {
			c.side[i] = tap[i]
		}
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
