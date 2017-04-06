package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Pan", func(Config) (Patcher, error) { return newPan() })
}

type pan struct {
	multiOutIO
	in, bias *In
	a, b     dsp.Frame
}

func newPan() (*pan, error) {
	m := &pan{
		in:   NewIn("input", dsp.Float64(0)),
		bias: NewInBuffer("bias", dsp.Float64(0)),
		a:    dsp.NewFrame(),
		b:    dsp.NewFrame(),
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

func (p *pan) Process(out dsp.Frame) {
	p.incrRead(func() {
		p.in.Process(out)
		bias := p.bias.ProcessFrame()
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
