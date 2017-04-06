package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Crossfeed", func(Config) (Patcher, error) { return newCrossfeed() })
}

type crossfeed struct {
	multiOutIO
	aIn, bIn, amount *In
	aOut, bOut       dsp.Frame
}

func newCrossfeed() (*crossfeed, error) {
	m := &crossfeed{
		aIn:    NewInBuffer("a", dsp.Float64(zero)),
		bIn:    NewInBuffer("b", dsp.Float64(zero)),
		amount: NewInBuffer("amount", dsp.Float64(zero)),
		aOut:   dsp.NewFrame(),
		bOut:   dsp.NewFrame(),
	}
	err := m.Expose(
		"Crossfeed",
		[]*In{m.aIn, m.bIn, m.amount},
		[]*Out{
			{Name: "a", Provider: provideCopyOut(m, &m.aOut)},
			{Name: "b", Provider: provideCopyOut(m, &m.bOut)},
		},
	)
	return m, err
}

func (p *crossfeed) Process(out dsp.Frame) {
	p.incrRead(func() {
		a := p.aIn.ProcessFrame()
		b := p.bIn.ProcessFrame()
		amount := p.amount.ProcessFrame()

		for i := range out {
			amt := dsp.Clamp(amount[i], 0, 1)
			p.aOut[i] = a[i] + (amt * b[i])
			p.bOut[i] = b[i] + (amt * a[i])
		}
	})
}
