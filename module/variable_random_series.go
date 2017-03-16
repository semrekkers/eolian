package module

func init() {
	Register("VariableRandomSeries", func(Config) (Patcher, error) { return newVariableRandomSeries() })
}

type variableRandomSeries struct {
	multiOutIO
	clock, size, random, min, max *In
	idx                           int
	memory, gateMemory            []Value

	value, gates Frame
	lastClock    Value
}

func newVariableRandomSeries() (*variableRandomSeries, error) {
	m := &variableRandomSeries{
		clock:      &In{Name: "clock", Source: NewBuffer(zero)},
		size:       &In{Name: "size", Source: NewBuffer(Value(8))},
		random:     &In{Name: "random", Source: NewBuffer(zero)},
		min:        &In{Name: "min", Source: NewBuffer(zero)},
		max:        &In{Name: "max", Source: NewBuffer(Value(1))},
		memory:     make([]Value, randomSeriesMax),
		gateMemory: make([]Value, randomSeriesMax),
		value:      make(Frame, FrameSize),
		gates:      make(Frame, FrameSize),
	}

	return m, m.Expose(
		"VariableRandomSeries",
		[]*In{m.size, m.random, m.clock, m.min, m.max},
		[]*Out{
			{Name: "value", Provider: provideCopyOut(m, &m.value)},
			{Name: "gate", Provider: provideCopyOut(m, &m.gates)},
		},
	)
}

func (s *variableRandomSeries) Read(out Frame) {
	s.incrRead(func() {
		var (
			clock  = s.clock.ReadFrame()
			size   = s.size.ReadFrame()
			random = s.random.ReadFrame()
			min    = s.min.ReadFrame()
			max    = s.max.ReadFrame()
		)

		for i := range out {
			size := clampValue(size[i], 1, randomSeriesMax)

			if s.lastClock < 0 && clock[i] > 0 {
				if r := random[i]; r != 0 && (r == 1 || randValue() > 1-r) {
					scale := randValue()
					s.memory[s.idx] = scale*(max[i]-min[i]) + min[i]
					if scale > 0.5 {
						s.gateMemory[s.idx] = 1
					} else {
						s.gateMemory[s.idx] = -1
					}
				}
				s.idx = (s.idx + 1) % int(size)
			}

			s.value[i] = s.memory[s.idx]
			if s.gateMemory[s.idx] > 0 {
				s.gates[i] = 1
			} else {
				s.gates[i] = -1
			}

			s.lastClock = clock[i]
		}
	})
}
