package module

func init() {
	Register("ADSR", func(Config) (Patcher, error) { return newADSR() })
}

type adsr struct {
	multiOutIO
	gate, ratio, disableSustain,
	attack, decay, release, sustain *In

	stateFunc            adsrStateFunc
	state                *adsrState
	mainOut, endCycleOut Frame
}

func newADSR() (*adsr, error) {
	m := &adsr{
		gate:           &In{Name: "gate", Source: NewBuffer(zero)},
		attack:         &In{Name: "attack", Source: NewBuffer(Duration(10))},
		decay:          &In{Name: "decay", Source: NewBuffer(Duration(10))},
		release:        &In{Name: "release", Source: NewBuffer(Duration(10))},
		ratio:          &In{Name: "ratio", Source: NewBuffer(Value(0.01))},
		sustain:        &In{Name: "sustain", Source: NewBuffer(Value(0.1))},
		disableSustain: &In{Name: "disableSustain", Source: NewBuffer(zero)},

		stateFunc:   adsrIdle,
		state:       &adsrState{lastGate: -1},
		mainOut:     make(Frame, FrameSize),
		endCycleOut: make(Frame, FrameSize),
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

func (e *adsr) Read(out Frame) {
	e.incrRead(func() {
		var (
			gate           = e.gate.ReadFrame()
			attack         = e.attack.ReadFrame()
			decay          = e.decay.ReadFrame()
			release        = e.release.ReadFrame()
			sustain        = e.sustain.ReadFrame()
			disableSustain = e.disableSustain.ReadFrame()
			ratio          = e.ratio.ReadFrame()
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

			if e.state.endOfCycle {
				e.endCycleOut[i] = 1
			} else {
				e.endCycleOut[i] = -1
			}
		}
	})
}

type adsrState struct {
	value, gate, attack, decay, sustain, disableSustain, release Value

	ratio            Value
	base, multiplier Value
	lastGate         Value
	endOfCycle       bool
}

type adsrStateFunc func(*adsrState) adsrStateFunc

func adsrIdle(s *adsrState) adsrStateFunc {
	s.endOfCycle = false
	if s.lastGate <= 0 && s.gate > 0 {
		s.value = 0
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
		s.endOfCycle = true
		if s.lastGate <= 0 && s.gate > 0 {
			return prepAttack(s)
		}
	} else {
		if s.gate > 0 {
			return prepAttack(s)
		}
	}
	s.value = s.base + s.value*s.multiplier
	if float64(s.value) <= epsilon {
		s.value = 0
		return adsrIdle
	}
	return adsrRelease
}

func prepAttack(s *adsrState) adsrStateFunc {
	s.endOfCycle = false
	s.multiplier = expRatio(s.ratio, s.attack)
	s.base = (1.0 + s.ratio) * (1.0 - s.multiplier)
	return adsrAttack
}

func prepDecay(s *adsrState) adsrStateFunc {
	s.multiplier = expRatio(s.ratio, s.decay)
	s.base = (s.sustain - s.ratio) * (1.0 - s.multiplier)
	return adsrDecay
}

func prepRelease(s *adsrState) adsrStateFunc {
	s.multiplier = expRatio(s.ratio, s.release)
	s.base = -s.ratio * (1.0 - s.multiplier)
	return adsrRelease
}
