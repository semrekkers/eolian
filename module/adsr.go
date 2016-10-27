package module

func init() {
	Register("ADSR", func(Config) (Patcher, error) { return NewADSR() })
}

type ADSR struct {
	IO
	gate, attack, decay, release *In
	sustain, disableSustain      *In

	stateFunc adsrStateFunc
	state     *adsrState
}

func NewADSR() (*ADSR, error) {
	m := &ADSR{
		gate:           &In{Name: "gate", Source: NewBuffer(zero)},
		attack:         &In{Name: "attack", Source: NewBuffer(Duration(10))},
		decay:          &In{Name: "decay", Source: NewBuffer(Duration(10))},
		release:        &In{Name: "release", Source: NewBuffer(Duration(10))},
		sustain:        &In{Name: "sustain", Source: NewBuffer(Value(0.1))},
		disableSustain: &In{Name: "disableSustain", Source: NewBuffer(zero)},

		stateFunc: adsrIdle,
		state:     &adsrState{},
	}
	err := m.Expose(
		[]*In{m.gate, m.attack, m.decay, m.release, m.sustain, m.disableSustain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *ADSR) Read(out Frame) {
	gate := reader.gate.ReadFrame()
	attack := reader.attack.ReadFrame()
	decay := reader.decay.ReadFrame()
	release := reader.release.ReadFrame()
	sustain := reader.sustain.ReadFrame()
	disableSustain := reader.disableSustain.ReadFrame()

	for i := range out {
		reader.state.lastGate = reader.state.gate
		reader.state.gate = gate[i]
		reader.state.attack = attack[i]
		reader.state.decay = decay[i]
		reader.state.sustain = sustain[i]
		reader.state.disableSustain = disableSustain[i]
		reader.state.release = release[i]
		reader.stateFunc = reader.stateFunc(reader.state)
		out[i] = reader.state.value
	}
}

type adsrState struct {
	value, gate, attack, decay, sustain, disableSustain, release Value
	start                                                        Value
	lastGate                                                     Value
}

type adsrStateFunc func(*adsrState) adsrStateFunc

func adsrIdle(s *adsrState) adsrStateFunc {
	if s.lastGate < 0 && s.gate > 0 {
		s.start = s.value
		return adsrAttack
	}
	return adsrIdle
}

func adsrAttack(s *adsrState) adsrStateFunc {
	s.value += (1 - s.start) / s.attack
	if s.value >= 1 {
		s.value = 1
		return adsrDecay
	}
	return adsrAttack
}

func adsrDecay(s *adsrState) adsrStateFunc {
	s.value -= (1 - s.sustain) / s.decay
	if s.value <= s.sustain {
		s.value = s.sustain
		if s.disableSustain == 1 {
			return adsrRelease
		}
		return adsrSustain
	}
	return adsrDecay
}

func adsrSustain(s *adsrState) adsrStateFunc {
	if s.gate < 0 {
		return adsrRelease
	}
	return adsrSustain
}

func adsrRelease(s *adsrState) adsrStateFunc {
	if s.disableSustain == 1 {
		if s.lastGate < 0 && s.gate > 0 {
			return adsrAttack
		}
	} else {
		if s.gate > 0 {
			s.start = s.value
			return adsrAttack
		}
	}
	s.value -= s.sustain / s.release
	if float64(s.value) <= epsilon {
		s.value = 0
		return adsrIdle
	}
	return adsrRelease
}
