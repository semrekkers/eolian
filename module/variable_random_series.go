package module

import "buddin.us/eolian/dsp"

func init() {
	Register("VariableRandomSeries", func(Config) (Patcher, error) { return newVariableRandomSeries() })
}

type variableRandomSeries struct {
	multiOutIO
	clock, size, random, min, max *In
	idx                           int
	memory, gateMemory            []dsp.Float64

	value, gates dsp.Frame
	lastClock    dsp.Float64
}

func newVariableRandomSeries() (*variableRandomSeries, error) {
	m := &variableRandomSeries{
		clock:      NewInBuffer("clock", dsp.Float64(0)),
		size:       NewInBuffer("size", dsp.Float64(8)),
		random:     NewInBuffer("random", dsp.Float64(0)),
		min:        NewInBuffer("min", dsp.Float64(0)),
		max:        NewInBuffer("max", dsp.Float64(1)),
		memory:     make([]dsp.Float64, randomSeriesMax),
		gateMemory: make([]dsp.Float64, randomSeriesMax),
		value:      dsp.NewFrame(),
		gates:      dsp.NewFrame(),
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

func (s *variableRandomSeries) Process(out dsp.Frame) {
	s.incrRead(func() {
		var (
			clock  = s.clock.ProcessFrame()
			size   = s.size.ProcessFrame()
			random = s.random.ProcessFrame()
			min    = s.min.ProcessFrame()
			max    = s.max.ProcessFrame()
		)

		for i := range out {
			size := dsp.Clamp(size[i], 1, randomSeriesMax)

			if s.lastClock < 0 && clock[i] > 0 {
				if r := random[i]; r != 0 && (r == 1 || dsp.Rand() > 1-r) {
					s.memory[s.idx] = dsp.Rand()*(max[i]-min[i]) + min[i]
					if dsp.Rand() > 0.25 {
						s.gateMemory[s.idx] = 1
					} else {
						s.gateMemory[s.idx] = -1
					}
				}
				s.idx = (s.idx + 1) % int(size)
			}

			s.value[i] = s.memory[s.idx]
			s.gates[i] = s.gateMemory[s.idx]
			s.lastClock = clock[i]
		}
	})
}
