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

func (d *dynamics) calcCoefs(clamp, relax dsp.Float64) {
	if clamp != d.lastClamp || d.lastClamp == -1 {
		if clamp == 0 {
			d.clampCoef = 0
		} else {
			d.clampCoef = dsp.Exp(log1 / clamp)
		}
		d.lastClamp = clamp
	}
	if relax != d.lastRelax || d.lastRelax == -1 {
		if relax == 0 {
			d.relaxCoef = 0
		} else {
			d.relaxCoef = dsp.Exp(log1 / relax)
		}
		d.lastRelax = relax
	}
}

func (d *dynamics) Process(out dsp.Frame) {
	d.in.Process(out)

	var (
		control   = d.control.ProcessFrame()
		threshold = d.threshold.ProcessFrame()[0]
		above     = d.above.ProcessFrame()[0]
		below     = d.below.ProcessFrame()[0]
		clamp     = d.clamp.ProcessFrame()[0]
		relax     = d.relax.ProcessFrame()[0]
	)

	d.calcCoefs(clamp, relax)

	for i := range out {
		v := dsp.Abs(control[i])
		if v < d.lastMax {
			v = v + (d.lastMax-v)*d.relaxCoef
		} else {
			v = v + (d.lastMax-v)*d.clampCoef
		}
		d.lastMax = v
	}

	var nextGain dsp.Float64
	if d.lastMax < threshold {
		if below == 1 {
			nextGain = 1
		} else {
			nextGain = dsp.Pow(d.lastMax/threshold, below-1)
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
			nextGain = dsp.Pow(d.lastMax/threshold, above-1)
		}
	}

	slope := (nextGain - d.lastGain) * slopeFactor
	for i := range out {
		out[i] = d.dcBlock.Tick(out[i] * d.lastGain)
		d.lastGain += slope
	}
}

func (d *dynamics) LuaState() map[string]interface{} {
	return map[string]interface{}{
		"rms": d.lastMax,
	}
}
