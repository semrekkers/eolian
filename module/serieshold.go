package module

func init() {
	Register("SeriesHold", func(Config) (Patcher, error) { return NewSeriesHold() })
}

type SeriesHold struct {
	IO
	in, clock, size, trigger *In

	state     *seriesHoldState
	stateFunc seriesHoldFunc
}

func NewSeriesHold() (*SeriesHold, error) {
	m := &SeriesHold{
		in:      &In{Name: "input", Source: zero},
		clock:   &In{Name: "clock", Source: NewBuffer(zero)},
		size:    &In{Name: "size", Source: NewBuffer(Value(8))},
		trigger: &In{Name: "trigger", Source: NewBuffer(zero)},
		state: &seriesHoldState{
			memory:      make([]Value, 32),
			lastTrigger: -1,
			lastClock:   -1,
		},
		stateFunc: seriesHoldLearn,
	}
	err := m.Expose(
		[]*In{m.in, m.size, m.trigger, m.clock},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *SeriesHold) Read(out Frame) {
	reader.in.Read(out)
	size := reader.size.ReadFrame()
	trigger := reader.trigger.ReadFrame()
	clock := reader.clock.ReadFrame()

	for i := range out {
		reader.state.lastClock = reader.state.clock
		reader.state.lastTrigger = reader.state.trigger
		reader.state.in = out[i]
		reader.state.size = clampValue(size[i], 1, 32)
		reader.state.trigger = trigger[i]
		reader.state.clock = clock[i]
		reader.stateFunc = reader.stateFunc(reader.state)
		out[i] = reader.state.value
	}
}

type seriesHoldState struct {
	value                    Value
	in, clock, size, trigger Value
	lastClock, lastTrigger   Value
	memory                   []Value

	idx int
}

type seriesHoldFunc func(*seriesHoldState) seriesHoldFunc

func seriesHoldNormal(s *seriesHoldState) seriesHoldFunc {
	if s.lastTrigger < 0 && s.trigger > 0 {
		return seriesHoldLearn
	}
	s.value = s.memory[s.idx]
	if s.lastClock < 0 && s.clock > 0 {
		s.idx++
		if s.idx >= int(s.size) {
			s.idx = 0
		}
	}
	return seriesHoldNormal
}

func seriesHoldLearn(s *seriesHoldState) seriesHoldFunc {
	if s.lastTrigger < 0 && s.trigger > 0 {
		s.idx = 0
	}
	if s.lastClock < 0 && s.clock > 0 {
		s.memory[s.idx] = s.in
		s.value = s.in
		s.idx++
		if s.idx >= int(s.size) {
			s.idx = 0
			return seriesHoldNormal
		}
	}
	s.value = s.memory[s.idx]
	return seriesHoldLearn
}
