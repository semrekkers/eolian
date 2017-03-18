package module

func init() {
	Register("Shape", func(Config) (Patcher, error) { return newShape() })
}

type shape struct {
	multiOutIO
	gate, trigger, rise, fall, cycle, ratio *In

	state          *shapeState
	stateFunc      shapeStateFunc
	main, endCycle Frame
}

func newShape() (*shape, error) {
	m := &shape{
		gate:    &In{Name: "gate", Source: NewBuffer(zero)},
		trigger: &In{Name: "trigger", Source: NewBuffer(zero)},
		rise:    &In{Name: "rise", Source: NewBuffer(Duration(1))},
		fall:    &In{Name: "fall", Source: NewBuffer(Duration(1))},
		cycle:   &In{Name: "cycle", Source: NewBuffer(zero)},
		ratio:   &In{Name: "ratio", Source: NewBuffer(Value(0.01))},
		state: &shapeState{
			lastGate:    -1,
			lastTrigger: -1,
		},
		stateFunc: shapeIdle,
		main:      make(Frame, FrameSize),
		endCycle:  make(Frame, FrameSize),
	}
	return m, m.Expose(
		"Shape",
		[]*In{m.gate, m.trigger, m.rise, m.fall, m.cycle, m.ratio},
		[]*Out{
			{Name: "output", Provider: provideCopyOut(m, &m.main)},
			{Name: "endCycle", Provider: provideCopyOut(m, &m.endCycle)},
		},
	)
}

func (s *shape) Read(out Frame) {
	s.incrRead(func() {
		var (
			gate    = s.gate.ReadFrame()
			trigger = s.trigger.ReadFrame()
			rise    = s.rise.ReadFrame()
			fall    = s.fall.ReadFrame()
			cycle   = s.cycle.ReadFrame()
			ratio   = s.ratio.ReadFrame()
		)
		for i := range out {
			s.state.gate = gate[i]
			s.state.trigger = trigger[i]
			s.state.rise = rise[i]
			s.state.fall = fall[i]
			s.state.cycle = cycle[i]
			s.state.ratio = ratio[i]
			s.stateFunc = s.stateFunc(s.state)

			if s.state.endCycle {
				s.endCycle[i] = 1
			} else {
				s.endCycle[i] = -1
			}
			s.main[i] = s.state.out
		}
	})
}

type shapeState struct {
	out, gate, trigger, rise, fall, cycle, ratio Value
	base, multiplier, lastGate, lastTrigger      Value
	endCycle                                     bool
}

type shapeStateFunc func(*shapeState) shapeStateFunc

func shapeIdle(s *shapeState) shapeStateFunc {
	s.endCycle = false
	if s.lastGate <= 0 && s.gate > 0 {
		s.out = 0
		return prepRise(s)
	}
	if s.lastTrigger <= 0 && s.trigger > 0 {
		s.out = 0
		return prepRise(s)
	}
	return shapeIdle
}

func shapeRise(s *shapeState) shapeStateFunc {
	s.endCycle = false
	s.out = s.base + s.out*s.multiplier
	if s.out >= 1 {
		s.out = 1
		if s.gate > 0 {
			return shapeRise
		}
		return prepFall(s)
	}
	return shapeRise
}

func shapeFall(s *shapeState) shapeStateFunc {
	if s.gate > 0 || s.trigger > 0 {
		return prepRise(s)
	}
	s.out = s.base + s.out*s.multiplier
	if float64(s.out) <= epsilon {
		s.endCycle = true
		s.out = 0
		if s.cycle > 0 {
			return prepRise(s)
		}
		return shapeIdle
	}
	return shapeFall
}

func prepRise(s *shapeState) shapeStateFunc {
	s.multiplier = expRatio(s.ratio, s.rise)
	s.base = (1.0 + s.ratio) * (1.0 - s.multiplier)
	return shapeRise
}

func prepFall(s *shapeState) shapeStateFunc {
	s.multiplier = expRatio(s.ratio, s.fall)
	s.base = -s.ratio * (1.0 - s.multiplier)
	return shapeFall
}
