package module

import "buddin.us/eolian/dsp"

func init() {
	Register("ChanceGate", func(Config) (Patcher, error) { return newChanceGate() })
}

type chanceGate struct {
	multiOutIO
	in, bias *In
	a, b     dsp.Frame
	flip     bool
	lastIn   dsp.Float64
}

func newChanceGate() (*chanceGate, error) {
	m := &chanceGate{
		in:   NewIn("input", dsp.Float64(0)),
		bias: NewInBuffer("bias", dsp.Float64(0)),
		a:    dsp.NewFrame(),
		b:    dsp.NewFrame(),
	}

	return m, m.Expose("ChanceGate", []*In{m.in, m.bias}, []*Out{
		{Name: "a", Provider: provideCopyOut(m, &m.a)},
		{Name: "b", Provider: provideCopyOut(m, &m.b)},
	})
}

func (c *chanceGate) Process(out dsp.Frame) {
	c.incrRead(func() {
		c.in.Process(out)
		bias := c.bias.ProcessFrame()
		for i := range out {
			if c.lastIn < 0 && out[i] > 0 {
				r := dsp.Rand()
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
