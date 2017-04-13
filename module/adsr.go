package module

import (
	"buddin.us/eolian/dsp"
)

func init() {
	Register("ADSR", func(Config) (Patcher, error) { return newADSR() })
}

type adsr struct {
	multiOutIO
	gate, ratio, disableSustain,
	attack, decay, release, sustain *In

	stateFunc            adsrStateFunc
	state                *adsrState
	mainOut, endCycleOut dsp.Frame
}

func newADSR() (*adsr, error) {
	m := &adsr{
		gate:           NewInBuffer("gate", dsp.Float64(0)),
		attack:         NewInBuffer("attack", dsp.Duration(10)),
		decay:          NewInBuffer("decay", dsp.Duration(10)),
		release:        NewInBuffer("release", dsp.Duration(10)),
		ratio:          NewInBuffer("ratio", dsp.Float64(0.01)),
		sustain:        NewInBuffer("sustain", dsp.Float64(0.1)),
		disableSustain: NewInBuffer("disableSustain", dsp.Float64(0)),
		stateFunc:      adsrIdle,
		state:          &adsrState{lastGate: -1},
		mainOut:        dsp.NewFrame(),
		endCycleOut:    dsp.NewFrame(),
	}
	err := m.Expose(
		"ADSR",
		[]*In{m.gate, m.attack, m.decay, m.release, m.sustain, m.disableSustain, m.ratio},
		[]*Out{
			{Name: "output", Provider: provideCopyOut(m, &m.mainOut)},
			{Name: "endCycle", Provider: provideCopyOut(m, &m.endCycleOut)},
		},
	)
	return m, err
}

func (e *adsr) Process(out dsp.Frame) {
	e.incrRead(func() {
		var (
			gate           = e.gate.ProcessFrame()
			attack         = e.attack.ProcessFrame()
			decay          = e.decay.ProcessFrame()
			release        = e.release.ProcessFrame()
			sustain        = e.sustain.ProcessFrame()
			disableSustain = e.disableSustain.ProcessFrame()
			ratio          = e.ratio.ProcessFrame()
		)

		for i := range out {
			e.state.lastGate = e.state.gate
			e.state.gate = gate[i]
			e.state.attack = attack[i]
			e.state.decay = decay[i]
			e.state.sustain = sustain[i]
			e.state.disableSustain = disableSustain[i]
			e.state.release = release[i]
			e.state.ratio = ratio[i]
			e.stateFunc = e.stateFunc(e.state)
			e.mainOut[i] = e.state.value

			if e.state.endCycle {
				e.endCycleOut[i] = 1
			} else {
				e.endCycleOut[i] = -1
			}
		}
	})
}

type adsrState struct {
	value, gate, attack, decay, sustain, disableSustain, release dsp.Float64

	ratio            dsp.Float64
	base, multiplier dsp.Float64
	lastGate         dsp.Float64
	endCycle         bool
}

type adsrStateFunc func(*adsrState) adsrStateFunc

func adsrIdle(s *adsrState) adsrStateFunc {
	s.endCycle = false
	s.value = 0
	if s.lastGate <= 0 && s.gate > 0 {
		return prepAttack(s)
	}
	return adsrIdle
}

func adsrAttack(s *adsrState) adsrStateFunc {
	s.value = s.base + s.value*s.multiplier
	if s.value >= 1 {
		if s.decay == 0 {
			s.value = s.sustain
			if s.disableSustain == 1 {
				return prepRelease(s)
			}
			return adsrSustain
		}
		s.value = 1
		return prepDecay(s)
	}
	return adsrAttack
}

func adsrDecay(s *adsrState) adsrStateFunc {
	s.value = s.base + s.value*s.multiplier
	if s.value <= s.sustain {
		s.value = s.sustain
		if s.disableSustain == 1 {
			return prepRelease(s)
		}
		return adsrSustain
	}
	return prepDecay(s)
}

func adsrSustain(s *adsrState) adsrStateFunc {
	if s.gate <= 0 {
		return prepRelease(s)
	}
	return adsrSustain
}

func adsrRelease(s *adsrState) adsrStateFunc {
	if s.disableSustain == 1 {
		s.endCycle = true
		if s.lastGate <= 0 && s.gate > 0 {
			return prepAttack(s)
		}
	} else {
		if s.gate > 0 {
			return prepAttack(s)
		}
	}
	s.value = s.base + s.value*s.multiplier
	if float64(s.value) <= dsp.Epsilon {
		s.value = 0
		return adsrIdle
	}
	return adsrRelease
}

func prepAttack(s *adsrState) adsrStateFunc {
	s.endCycle = false
	s.base, s.multiplier = shapeCoeffs(s.ratio, s.attack, 1, logCurve)
	return adsrAttack
}

func prepDecay(s *adsrState) adsrStateFunc {
	s.base, s.multiplier = shapeCoeffs(s.ratio, s.decay, s.sustain, expCurve)
	return adsrDecay
}

func prepRelease(s *adsrState) adsrStateFunc {
	s.base, s.multiplier = shapeCoeffs(s.ratio, s.release, 0, expCurve)
	return adsrRelease
}

const (
	expCurve int = iota
	logCurve
)

func shapeCoeffs(ratio, duration, target dsp.Float64, curve int) (base, multiplier dsp.Float64) {
	multiplier = dsp.ExpRatio(ratio, duration)
	if curve == expCurve {
		ratio = -ratio
	}
	base = (target + ratio) * (1.0 - multiplier)
	return
}
