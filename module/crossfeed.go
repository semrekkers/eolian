package module

func init() {
	Register("Crossfeed", func(Config) (Patcher, error) { return newCrossfeed() })
}

type crossfeed struct {
	multiOutIO
	aIn, bIn, amount *In
	aOut, bOut       Frame
}

func newCrossfeed() (*crossfeed, error) {
	m := &crossfeed{
		aIn:    &In{Name: "a", Source: NewBuffer(zero)},
		bIn:    &In{Name: "b", Source: NewBuffer(zero)},
		amount: &In{Name: "amount", Source: NewBuffer(zero)},
		aOut:   make(Frame, FrameSize),
		bOut:   make(Frame, FrameSize),
	}
	err := m.Expose(
		"crossfeed",
		[]*In{m.aIn, m.bIn, m.amount},
		[]*Out{
			{Name: "a", Provider: provideCopyOut(m, &m.aOut)},
			{Name: "b", Provider: provideCopyOut(m, &m.bOut)},
		},
	)
	return m, err
}

func (p *crossfeed) Read(out Frame) {
	p.incrRead(func() {
		a := p.aIn.ReadFrame()
		b := p.bIn.ReadFrame()
		amount := p.amount.ReadFrame()

		for i := range out {
			amt := clampValue(amount[i], 0, 1)
			p.aOut[i] = a[i] + (amt * b[i])
			p.bOut[i] = b[i] + (amt * a[i])
		}
	})
}
