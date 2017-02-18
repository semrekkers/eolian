package module

import "math/rand"

func init() {
	Register("RandomSeries", func(Config) (Patcher, error) { return newRandomSeries() })
}

const randomSeriesMax = 32

type randomSeries struct {
	IO
	clock, size, trigger, min, max   *In
	idx                              int
	memory, gateMemory               []Value
	lastSize, lastTrigger, lastClock Value

	readTracker        manyReadTracker
	valuesOut, gateOut Frame
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
		valuesOut:   make(Frame, FrameSize),
		gateOut:     make(Frame, FrameSize),
		lastTrigger: -1,
		lastClock:   -1,
	}

	m.readTracker = manyReadTracker{counter: m}

	return m, m.Expose(
		"RandomSeries",
		[]*In{m.size, m.trigger, m.clock, m.min, m.max},
		[]*Out{
			{Name: "values", Provider: m.out(&m.valuesOut)},
			{Name: "gate", Provider: m.out(&m.gateOut)},
		},
	)
}

func (s *randomSeries) out(cache *Frame) ReaderProvider {
	return ReaderProviderFunc(func() Reader {
		return &manyOut{reader: s, cache: cache}
	})
}

func (s *randomSeries) readMany(out Frame) {
	if s.readTracker.count() > 0 {
		s.readTracker.incr()
		return
	}

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
			for i := 0; i < int(size); i++ {
				s.memory[i] = randValue()*(max[i]-min[i]) + min[i]
				if rand.Float32() > 0.25 {
					s.gateMemory[i] = 1
				} else {
					s.gateMemory[i] = -1
				}
			}
		}

		s.valuesOut[i] = s.memory[s.idx]
		if s.gateMemory[s.idx] > 0 {
			s.gateOut[i] = clock[i]
		} else {
			s.gateOut[i] = -1
		}

		s.lastSize = size
		s.lastTrigger = trigger[i]
		s.lastClock = clock[i]
	}

	s.readTracker.incr()
}
