package module

func init() {
	Register("ChanceGate", func(Config) (Patcher, error) { return newChanceGate() })
}

type chanceGate struct {
	multiOutIO
	in, bias *In
	a, b     Frame
	flip     bool
	lastIn   Value
}

func newChanceGate() (*chanceGate, error) {
	m := &chanceGate{
		in:   &In{Name: "input", Source: zero},
		bias: &In{Name: "bias", Source: NewBuffer(zero)},
		a:    make(Frame, FrameSize),
		b:    make(Frame, FrameSize),
	}

	return m, m.Expose("ChanceGate", []*In{m.in, m.bias}, []*Out{
		{Name: "a", Provider: provideCopyOut(m, &m.a)},
		{Name: "b", Provider: provideCopyOut(m, &m.b)},
	})
}

func (c *chanceGate) Read(out Frame) {
	c.incrRead(func() {
		c.in.Read(out)
		bias := c.bias.ReadFrame()
		for i := range out {
			if c.lastIn < 0 && out[i] > 0 {
				r := randValue()
				if r < 0.5*(bias[i]+1) {
					c.flip = true
				} else {
					c.flip = false
				}
			}

			if c.flip {
				c.a[i] = out[i]
				c.b[i] = -1
			} else {
				c.a[i] = -1
				c.b[i] = out[i]
			}
			c.lastIn = out[i]
		}
	})
}
