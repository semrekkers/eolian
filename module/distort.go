package module

import "math"

func init() {
	Register("Distort", func(Config) (Patcher, error) { return NewDistort() })
}

type Distort struct {
	IO
	in, gain         *In
	offsetA, offsetB *In

	dcBlock *DCBlock

	lastIn, lastOut Value
}

func NewDistort() (*Distort, error) {
	m := &Distort{
		in:      &In{Name: "input", Source: zero},
		gain:    &In{Name: "gain", Source: NewBuffer(Value(1))},
		offsetA: &In{Name: "offsetA", Source: NewBuffer(zero)},
		offsetB: &In{Name: "offsetB", Source: NewBuffer(zero)},
		dcBlock: &DCBlock{},
	}
	err := m.Expose(
		"Distort",
		[]*In{m.in, m.offsetA, m.offsetB, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (d *Distort) Read(out Frame) {
	d.in.Read(out)
	offsetA, offsetB := d.offsetA.ReadFrame(), d.offsetB.ReadFrame()
	gain := d.gain.ReadFrame()

	var num, denom float64
	for i := range out {
		num = math.Exp(float64(out[i]*(offsetA[i]+gain[i]))) -
			math.Exp(float64(-out[i]*(offsetB[i]+gain[i])))
		denom = math.Exp(float64(out[i]*gain[i])) +
			math.Exp(float64(out[i]*-gain[i]))

		out[i] = d.dcBlock.Tick(Value(num / denom))
	}
}
