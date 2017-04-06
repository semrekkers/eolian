package module

import (
	"buddin.us/eolian/dsp"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Glide", func(c Config) (Patcher, error) {
		var config struct {
			Rise, Fall float64
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

func newGlide(rise, fall float64) (*glide, error) {
	m := &glide{
		in:   NewIn("input", dsp.Float64(0)),
		rise: NewInBuffer("rise", dsp.Duration(rise)),
		fall: NewInBuffer("fall", dsp.Duration(fall)),
		slew: newSlew(),
	}
	err := m.Expose(
		"Glide",
		[]*In{m.in, m.rise, m.fall},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (g *glide) Process(out dsp.Frame) {
	g.in.Process(out)
	rise, fall := g.rise.ProcessFrame(), g.fall.ProcessFrame()
	for i := range out {
		out[i] = g.slew.Tick(out[i], rise[i], fall[i])
	}
}

type slewStateFunc func(*slewState) slewStateFunc

type slewState struct {
	value, in, lastIn dsp.Float64
	from, to          dsp.Float64
	rise, fall        dsp.Float64
}

type slew struct {
	stateFunc slewStateFunc
	state     *slewState
}

func newSlew() *slew {
	return &slew{slewIdle, &slewState{}}
}

func (s *slew) Tick(v, rise, fall dsp.Float64) dsp.Float64 {
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
	if s.in != s.lastIn && dsp.Abs(s.in-s.lastIn) > dsp.Float64(dsp.Epsilon) {
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
		amount dsp.Float64
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
	} else if dsp.Abs(d) < dsp.Float64(dsp.Epsilon) {
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
