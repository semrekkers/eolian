package module

func init() {
	Register("Pan", func(Config) (Patcher, error) { return newPan() })
}

type pan struct {
	multiOutIO
	in, bias *In
	a, b     Frame
}

func newPan() (*pan, error) {
	m := &pan{
		in:   &In{Name: "input", Source: zero},
		bias: &In{Name: "bias", Source: NewBuffer(zero)},
		a:    make(Frame, FrameSize),
		b:    make(Frame, FrameSize),
	}
	err := m.Expose(
		"Pan",
		[]*In{m.in, m.bias},
		[]*Out{
			{Name: "a", Provider: provideCopyOut(m, &m.a)},
			{Name: "b", Provider: provideCopyOut(m, &m.b)},
		},
	)
	return m, err
}

func (p *pan) Read(out Frame) {
	p.incrRead(func() {
		p.in.Read(out)
		bias := p.bias.ReadFrame()
		for i := range out {
			if bias[i] > 0 {
				p.a[i] = (1 - bias[i]) * out[i]
				p.b[i] = out[i]
			} else if bias[i] < 0 {
				p.a[i] = out[i]
				p.b[i] = (1 + bias[i]) * out[i]
			} else {
				p.a[i] = out[i]
				p.b[i] = out[i]
			}
		}
	})
}
