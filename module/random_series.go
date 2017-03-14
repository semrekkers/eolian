package module

import "math/rand"

func init() {
	Register("RandomSeries", func(Config) (Patcher, error) { return newRandomSeries() })
}

const randomSeriesMax = 32

type randomSeries struct {
	multiOutIO
	clock, size, trigger, min, max *In
	idx                            int
	memory, gateMemory             []Value
	lastTrigger, lastClock         Value

	valueOut, gateOut Frame
}

func newRandomSeries() (*randomSeries, error) {
	m := &randomSeries{
		clock:       &In{Name: "clock", Source: NewBuffer(zero)},
		size:        &In{Name: "size", Source: NewBuffer(Value(8))},
		trigger:     &In{Name: "trigger", Source: NewBuffer(zero)},
		min:         &In{Name: "min", Source: NewBuffer(zero)},
		max:         &In{Name: "max", Source: NewBuffer(Value(1))},
		memory:      make([]Value, randomSeriesMax),
		gateMemory:  make([]Value, randomSeriesMax),
		valueOut:    make(Frame, FrameSize),
		gateOut:     make(Frame, FrameSize),
		lastTrigger: -1,
		lastClock:   -1,
	}

	return m, m.Expose(
		"RandomSeries",
		[]*In{m.size, m.trigger, m.clock, m.min, m.max},
		[]*Out{
			{Name: "value", Provider: provideCopyOut(m, &m.valueOut)},
			{Name: "gate", Provider: provideCopyOut(m, &m.gateOut)},
		},
	)
}

func (s *randomSeries) Read(out Frame) {
	s.incrRead(func() {
		var (
			min     = s.min.ReadFrame()
			max     = s.max.ReadFrame()
			trigger = s.trigger.ReadFrame()
			size    = s.size.ReadFrame()
			clock   = s.clock.ReadFrame()
		)

		for i := range out {
			size := clampValue(size[i], 1, randomSeriesMax)
			if s.lastClock < 0 && clock[i] > 0 {
				s.idx++
				if s.idx >= int(size) {
					s.idx = 0
				}
			}
			if s.lastTrigger < 0 && trigger[i] > 0 {
				for j := 0; j < int(size); j++ {
					s.memory[j] = randValue()*(max[j]-min[j]) + min[j]
					if rand.Float32() > 0.25 {
						s.gateMemory[j] = 1
					} else {
						s.gateMemory[j] = -1
					}
				}
			}

			s.valueOut[i] = s.memory[s.idx]
			if s.gateMemory[s.idx] > 0 {
				s.gateOut[i] = clock[i]
			} else {
				s.gateOut[i] = -1
			}

			s.lastTrigger = trigger[i]
			s.lastClock = clock[i]
		}
	})
}
