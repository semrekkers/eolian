package module

import (
	"math"

	"buddin.us/eolian/dsp"
)

func init() {
	Register("Distort", func(Config) (Patcher, error) { return newDistort() })
}

type distort struct {
	IO
	in, gain         *In
	offsetA, offsetB *In

	dcBlock *dsp.DCBlock
}

func newDistort() (*distort, error) {
	m := &distort{
		in:      NewIn("input", dsp.Float64(0)),
		gain:    NewInBuffer("gain", dsp.Float64(1)),
		offsetA: NewInBuffer("offsetA", dsp.Float64(0)),
		offsetB: NewInBuffer("offsetB", dsp.Float64(0)),
		dcBlock: &dsp.DCBlock{},
	}
	err := m.Expose(
		"Distort",
		[]*In{m.in, m.offsetA, m.offsetB, m.gain},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (d *distort) Process(out dsp.Frame) {
	d.in.Process(out)
	offsetA, offsetB := d.offsetA.ProcessFrame(), d.offsetB.ProcessFrame()
	gain := d.gain.ProcessFrame()

	var num, denom float64
	for i := range out {
		num = math.Exp(float64(out[i]*(offsetA[i]+gain[i]))) -
			math.Exp(float64(-out[i]*(offsetB[i]+gain[i])))
		denom = math.Exp(float64(out[i]*gain[i])) +
			math.Exp(float64(out[i]*-gain[i]))

		out[i] = d.dcBlock.Tick(dsp.Float64(num / denom))
	}
}
