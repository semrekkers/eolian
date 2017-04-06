package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Shape", func(Config) (Patcher, error) { return newShape() })
}

type shape struct {
	multiOutIO
	gate, trigger, rise, fall, cycle, ratio *In

	state          *shapeState
	stateFunc      shapeStateFunc
	main, endCycle dsp.Frame
}

func newShape() (*shape, error) {
	m := &shape{
		gate:    NewInBuffer("gate", dsp.Float64(0)),
		trigger: NewInBuffer("trigger", dsp.Float64(0)),
		rise:    NewInBuffer("rise", dsp.Duration(1)),
		fall:    NewInBuffer("fall", dsp.Duration(1)),
		cycle:   NewInBuffer("cycle", dsp.Float64(0)),
		ratio:   NewInBuffer("ratio", dsp.Float64(0.01)),
		state: &shapeState{
			lastGate:    -1,
			lastTrigger: -1,
		},
		stateFunc: shapeIdle,
		main:      dsp.NewFrame(),
		endCycle:  dsp.NewFrame(),
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

func (s *shape) Process(out dsp.Frame) {
	s.incrRead(func() {
		var (
			gate    = s.gate.ProcessFrame()
			trigger = s.trigger.ProcessFrame()
			rise    = s.rise.ProcessFrame()
			fall    = s.fall.ProcessFrame()
			cycle   = s.cycle.ProcessFrame()
			ratio   = s.ratio.ProcessFrame()
		)
		for i := range out {
			s.state.lastGate = s.state.gate
			s.state.lastTrigger = s.state.trigger
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
	out, gate, trigger, rise, fall, cycle, ratio dsp.Float64
	base, multiplier, lastGate, lastTrigger      dsp.Float64
	endCycle                                     bool
}

type shapeStateFunc func(*shapeState) shapeStateFunc

func shapeIdle(s *shapeState) shapeStateFunc {
	s.endCycle = false
	s.out = 0
	if s.lastGate <= 0 && s.gate > 0 {
		return prepRise(s)
	}
	if s.lastTrigger <= 0 && s.trigger > 0 {
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
	if (s.lastGate <= 0 && s.gate > 0) || (s.lastTrigger <= 0 && s.trigger > 0) {
		return prepRise(s)
	}
	s.out = s.base + s.out*s.multiplier
	if float64(s.out) <= dsp.Epsilon {
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
	s.base, s.multiplier = shapeCoeffs(s.ratio, s.rise, 1, logCurve)
	return shapeRise
}

func prepFall(s *shapeState) shapeStateFunc {
	s.base, s.multiplier = shapeCoeffs(s.ratio, s.fall, 0, expCurve)
	return shapeFall
}
