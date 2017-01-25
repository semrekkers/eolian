package module

func init() {
	Register("Glide", func(Config) (Patcher, error) { return NewGlide() })
}

type Glide struct {
	IO
	in, rise, fall *In
	*slew
}

func NewGlide() (*Glide, error) {
	m := &Glide{
		in:   &In{Name: "input", Source: zero},
		rise: &In{Name: "rise", Source: NewBuffer(zero)},
		fall: &In{Name: "fall", Source: NewBuffer(zero)},
		slew: newSlew(),
	}
	err := m.Expose(
		"Glide",
		[]*In{m.in, m.rise, m.fall},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (g *Glide) Read(out Frame) {
	g.in.Read(out)
	rise, fall := g.rise.ReadFrame(), g.fall.ReadFrame()
	for i := range out {
		out[i] = g.slew.Tick(out[i], rise[i], fall[i])
	}
}

type slewStateFunc func(*slewState) slewStateFunc

type slewState struct {
	value, in, lastIn Value
	from, to          Value
	rise, fall        Value
}

type slew struct {
	stateFunc slewStateFunc
	state     *slewState
}

func newSlew() *slew {
	return &slew{slewIdle, &slewState{}}
}

func (s *slew) Tick(v, rise, fall Value) Value {
	s.state.lastIn, s.state.in = s.state.in, v
	s.state.rise, s.state.fall = rise, fall
	s.stateFunc = s.stateFunc(s.state)
	return s.state.value
}

func slewIdle(s *slewState) slewStateFunc {
	if s.lastIn == 0 && s.in != 0 {
		s.value = s.in
		return slewIdle
	}
	if s.in != s.lastIn && absValue(s.in-s.lastIn) > Value(epsilon) {
		s.from, s.to = s.lastIn, s.in
		s.lastIn = s.in
		s.value = s.from
		return slewTransition
	}
	return slewIdle
}

func slewTransition(s *slewState) slewStateFunc {
	if s.in != s.lastIn {
		s.from, s.to = s.lastIn, s.in
	}
	var (
		d      = s.to - s.from
		amount Value
	)
	if d < 0 {
		if s.fall == 0 {
			return slewFinish
		}
		amount = d / s.fall
	} else if d > 0 {
		if s.rise == 0 {
			return slewFinish
		}
		amount = d / s.rise
	} else if absValue(d) < Value(epsilon) {
		return slewFinish
	} else {
		return slewFinish
	}

	s.value += amount
	remain := s.value - s.to
	if (d > 0 && remain >= 0) || (d < 0 && remain <= 0) {
		return slewFinish
	}
	return slewTransition
}

func slewFinish(s *slewState) slewStateFunc {
	s.value = s.to
	return slewIdle
}
