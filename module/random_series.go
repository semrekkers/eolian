package module

import "math/rand"

func init() {
	Register("RandomSeries", func(Config) (Patcher, error) { return NewRandomSeries() })
}

const randomSeriesMax = 32

type RandomSeries struct {
	IO
	clock, size, trigger *In
	min, max             *In

	lastSize, lastTrigger, lastClock Value
	memory, gateMemory               []Value
	idx, reads                       int
}

func NewRandomSeries() (*RandomSeries, error) {
	m := &RandomSeries{
		clock:       &In{Name: "clock", Source: NewBuffer(zero)},
		size:        &In{Name: "size", Source: NewBuffer(Value(8))},
		trigger:     &In{Name: "trigger", Source: NewBuffer(zero)},
		min:         &In{Name: "min", Source: NewBuffer(zero)},
		max:         &In{Name: "max", Source: NewBuffer(Value(1))},
		memory:      make([]Value, randomSeriesMax),
		gateMemory:  make([]Value, randomSeriesMax),
		lastTrigger: -1,
		lastClock:   -1,
	}
	err := m.Expose(
		[]*In{m.size, m.trigger, m.clock, m.min, m.max},
		[]*Out{
			{
				Name: "values",
				Provider: ReaderProviderFunc(func() Reader {
					return &randomSeriesOut{RandomSeries: m}
				}),
			},
			{
				Name: "gate",
				Provider: ReaderProviderFunc(func() Reader {
					return &randomSeriesGate{RandomSeries: m}
				}),
			},
		},
	)
	return m, err
}

func (s *RandomSeries) read(out Frame) {
	if s.reads == 0 {
		s.min.ReadFrame()
		s.max.ReadFrame()

		trigger := s.trigger.ReadFrame()
		size := s.size.ReadFrame()
		clock := s.clock.ReadFrame()

		for i := range out {
			size := clampValue(size[i], 1, randomSeriesMax)
			if s.lastClock < 0 && clock[i] > 0 {
				s.idx++
				if s.idx >= int(size) {
					s.idx = 0
				}
			}
			s.lastSize = size
			s.lastTrigger = trigger[i]
			s.lastClock = clock[i]
		}
	}
	if outs := s.OutputsActive(); outs > 0 {
		s.reads = (s.reads + 1) % outs
	}
}

type randomSeriesOut struct {
	*RandomSeries
}

func (reader *randomSeriesOut) Read(out Frame) {
	reader.read(out)
	size := reader.size.LastFrame()
	trigger := reader.trigger.LastFrame()
	min := reader.min.LastFrame()
	max := reader.max.LastFrame()

	for i := range out {
		if (reader.lastTrigger < 0 && trigger[i] > 0) || (reader.lastSize != size[i]) {
			size := clampValue(size[i], 1, randomSeriesMax)
			for i := 0; i < int(size); i++ {
				reader.memory[i] = randValue()*(max[i]-min[i]) + min[i]
			}
		}
		out[i] = reader.memory[reader.idx]
	}
}

type randomSeriesGate struct {
	*RandomSeries
}

func (reader *randomSeriesGate) Read(out Frame) {
	reader.read(out)
	size := reader.size.LastFrame()
	trigger := reader.trigger.LastFrame()
	clock := reader.clock.LastFrame()

	for i := range out {
		size := clampValue(size[i], 1, randomSeriesMax)
		if reader.lastTrigger < 0 && trigger[i] > 0 {
			for i := 0; i < int(size); i++ {
				if rand.Float32() > 0.25 {
					reader.gateMemory[i] = 1
				} else {
					reader.gateMemory[i] = -1
				}
			}
		}
		if reader.gateMemory[reader.idx] > 0 {
			out[i] = clock[i]
		} else {
			out[i] = -1
		}
	}
}
