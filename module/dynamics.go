package module

import (
	"buddin.us/eolian/dsp"
)

func init() {
	Register("Dynamics", func(Config) (Patcher, error) { return newDynamics() })
}

var (
	slopeFactor = 1 / dsp.Float64(dsp.FrameSize)
	log1        = dsp.Log(0.1)
)

type dynamics struct {
	IO
	in, control, threshold, clamp, relax, above, below *In

	clampCoef, relaxCoef dsp.Float64
	lastClamp, lastRelax dsp.Float64
	lastGain, lastMax    dsp.Float64

	dcBlock *dsp.DCBlock
}

func newDynamics() (*dynamics, error) {
	m := &dynamics{
		in:        NewIn("input", dsp.Float64(0)),
		control:   NewInBuffer("control", dsp.Float64(0)),
		threshold: NewInBuffer("threshold", dsp.Float64(0.5)),
		above:     NewInBuffer("slopeAbove", dsp.Float64(0.3)),
		below:     NewInBuffer("slopeBelow", dsp.Float64(1)),
		clamp:     NewInBuffer("clamp", dsp.Duration(10)),
		relax:     NewInBuffer("relax", dsp.Duration(10)),
		dcBlock:   &dsp.DCBlock{},
		lastClamp: -1,
		lastRelax: -1,
	}
	err := m.Expose(
		"Dynamics",
		[]*In{m.in, m.clamp, m.relax, m.control, m.threshold, m.above, m.below},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (c *dynamics) calcCoefs(clamp, relax dsp.Float64) {
	if clamp != c.lastClamp || c.lastClamp == -1 {
		if clamp == 0 {
			c.clampCoef = 0
		} else {
			c.clampCoef = dsp.Exp(log1 / clamp)
		}
		c.lastClamp = clamp
	}
	if relax != c.lastRelax || c.lastRelax == -1 {
		if relax == 0 {
			c.relaxCoef = 0
		} else {
			c.relaxCoef = dsp.Exp(log1 / relax)
		}
		c.lastRelax = relax
	}
}

func (c *dynamics) Process(out dsp.Frame) {
	c.in.Process(out)

	var (
		control   = c.control.ProcessFrame()
		threshold = c.threshold.ProcessFrame()[0]
		above     = c.above.ProcessFrame()[0]
		below     = c.below.ProcessFrame()[0]
		clamp     = c.clamp.ProcessFrame()[0]
		relax     = c.relax.ProcessFrame()[0]
	)

	c.calcCoefs(clamp, relax)

	for i := range out {
		v := dsp.Abs(control[i])
		if v < c.lastMax {
			v = v + (c.lastMax-v)*c.relaxCoef
		} else {
			v = v + (c.lastMax-v)*c.clampCoef
		}
		c.lastMax = v
	}

	var nextGain dsp.Float64
	if c.lastMax < threshold {
		if below == 1 {
			nextGain = 1
		} else {
			nextGain = dsp.Pow(c.lastMax/threshold, below-1)
			absGain := dsp.Abs(nextGain)
			if absGain < 1.0e-15 {
				nextGain = 0
			} else if absGain > 1.0e15 {
				nextGain = 1
			}
		}
	} else {
		if above == 1 {
			nextGain = 1
		} else {
			nextGain = dsp.Pow(c.lastMax/threshold, above-1)
		}
	}

	slope := (nextGain - c.lastGain) * slopeFactor
	for i := range out {
		out[i] = c.dcBlock.Tick(out[i] * c.lastGain)
		c.lastGain += slope
	}
}
