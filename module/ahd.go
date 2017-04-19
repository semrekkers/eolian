package module

import (
	"math"

	"buddin.us/eolian/dsp"
)

func init() {
	Register("AHD", func(Config) (Patcher, error) { return newAHD() })
}

type ahd struct {
	multiOutIO
	gate, ratio, attack, decay, hold *In

	stateFunc            ahdStateFunc
	state                *ahdState
	mainOut, endCycleOut dsp.Frame
}

func newAHD() (*ahd, error) {
	m := &ahd{
		gate:        NewInBuffer("gate", dsp.Float64(0)),
		attack:      NewInBuffer("attack", dsp.Duration(10)),
		decay:       NewInBuffer("decay", dsp.Duration(10)),
		ratio:       NewInBuffer("ratio", dsp.Float64(0.01)),
		hold:        NewInBuffer("hold", dsp.Duration(50)),
		stateFunc:   ahdIdle,
		state:       &ahdState{lastGate: -1},
		mainOut:     dsp.NewFrame(),
		endCycleOut: dsp.NewFrame(),
	}
	err := m.Expose(
		"AHD",
		[]*In{m.gate, m.attack, m.decay, m.hold, m.ratio},
		[]*Out{
			{Name: "output", Provider: provideCopyOut(m, &m.mainOut)},
			{Name: "endcycle", Provider: provideCopyOut(m, &m.endCycleOut)},
		},
	)
	return m, err
}

func (e *ahd) Process(out dsp.Frame) {
	e.incrRead(func() {
		var (
			gate   = e.gate.ProcessFrame()
			attack = e.attack.ProcessFrame()
			decay  = e.decay.ProcessFrame()
			hold   = e.hold.ProcessFrame()
			ratio  = e.ratio.ProcessFrame()
		)

		for i := range out {
			e.state.lastGate = e.state.gate
			e.state.gate = gate[i]
			e.state.attack = attack[i]
			e.state.decay = decay[i]
			e.state.hold = hold[i]
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

type ahdState struct {
	value, gate, attack, decay, hold dsp.Float64

	holdTick int

	ratio            dsp.Float64
	base, multiplier dsp.Float64
	lastGate         dsp.Float64
	endCycle         bool
}

type ahdStateFunc func(*ahdState) ahdStateFunc

func ahdIdle(s *ahdState) ahdStateFunc {
	s.endCycle = false
	s.value = 0
	if s.lastGate <= 0 && s.gate > 0 {
		return prepAHDAttack(s)
	}
	return ahdIdle
}

func ahdAttack(s *ahdState) ahdStateFunc {
	s.value = s.base + s.value*s.multiplier
	if s.value >= 1 {
		s.value = 1
		return prepAHDDecay(s)
	}
	return ahdAttack
}

func ahdHold(s *ahdState) ahdStateFunc {
	s.holdTick++
	if s.holdTick >= int(s.hold) {
		s.holdTick = 0
		return prepAHDDecay
	}
	return ahdHold
}

func ahdDecay(s *ahdState) ahdStateFunc {
	if s.gate > 0 {
		return prepAHDAttack(s)
	}
	s.value = s.base + s.value*s.multiplier
	if float64(s.value) <= math.SmallestNonzeroFloat64 {
		s.value = 0
		return ahdIdle
	}
	return ahdDecay
}

func prepAHDAttack(s *ahdState) ahdStateFunc {
	s.endCycle = false
	s.base, s.multiplier = shapeCoeffs(s.ratio, s.attack, 1, logCurve)
	return ahdAttack
}

func prepAHDDecay(s *ahdState) ahdStateFunc {
	s.base, s.multiplier = shapeCoeffs(s.ratio, s.decay, 0, expCurve)
	return ahdDecay
}
