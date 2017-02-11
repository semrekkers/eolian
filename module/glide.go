package module

import "github.com/mitchellh/mapstructure"

func init() {
	Register("Glide", func(c Config) (Patcher, error) {
		var config struct {
			Rise, Fall int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Rise == 0 {
			config.Rise = 5
		}
		if config.Fall == 0 {
			config.Fall = 5
		}
		return newGlide(config.Rise, config.Fall)
	})
}

type glide struct {
	IO
	in, rise, fall *In
	*slew
}

func newGlide(rise, fall int) (*glide, error) {
	m := &glide{
		in:   &In{Name: "input", Source: zero},
		rise: &In{Name: "rise", Source: NewBuffer(DurationInt(rise))},
		fall: &In{Name: "fall", Source: NewBuffer(DurationInt(fall))},
		slew: newSlew(),
	}
	err := m.Expose(
		"Glide",
		[]*In{m.in, m.rise, m.fall},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (g *glide) Read(out Frame) {
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
