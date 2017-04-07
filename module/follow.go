package module

import (
	"math"

	"buddin.us/eolian/dsp"
)

func init() {
	Register("Follow", func(Config) (Patcher, error) { return newFollow() })
}

type follow struct {
	IO
	in, attack, release *In

	envelope dsp.Float64
}

func newFollow() (*follow, error) {
	m := &follow{
		in:      NewIn("input", dsp.Float64(0)),
		attack:  NewInBuffer("attack", dsp.Duration(10)),
		release: NewInBuffer("release", dsp.Duration(500)),
	}
	err := m.Expose(
		"Follow",
		[]*In{m.in, m.attack, m.release},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (c *follow) Process(out dsp.Frame) {
	c.in.Process(out)
	attack, release := c.attack.ProcessFrame(), c.release.ProcessFrame()
	for i := range out {
		in := dsp.Abs(out[i])
		side := release[i]
		if in > c.envelope {
			side = attack[i]
		}
		factor := math.Pow(0.01, float64(1.0/side))
		c.envelope = dsp.Float64(factor)*(c.envelope-in) + in
		out[i] = c.envelope
	}
}
