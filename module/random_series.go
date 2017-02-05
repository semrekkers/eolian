package module

import "math/rand"

func init() {
	Register("RandomSeries", func(Config) (Patcher, error) { return newRandomSeries() })
}

const randomSeriesMax = 32

type randomSeries struct {
	IO
	clock, size, trigger             *In
	min, max                         *In
	reads, idx                       int
	memory, gateMemory               []Value
	lastSize, lastTrigger, lastClock Value
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
		lastTrigger: -1,
		lastClock:   -1,
	}
	return m, m.Expose(
		"RandomSeries",
		[]*In{m.size, m.trigger, m.clock, m.min, m.max},
		[]*Out{
			{Name: "values", Provider: Provide(&randomSeriesOut{randomSeries: m})},
			{Name: "gate", Provider: Provide(&randomSeriesGate{randomSeries: m})},
		},
	)
}

func (s *randomSeries) read() {
	if s.reads > 0 {
		return
	}
	s.min.ReadFrame()
	s.max.ReadFrame()
	s.trigger.ReadFrame()
	s.size.ReadFrame()
	s.clock.ReadFrame()
}

func (s *randomSeries) postRead() {
	if outs := s.OutputsActive(); outs > 0 {
		s.reads = (s.reads + 1) % outs
	}
}

func (s *randomSeries) tick(times int, op func(int)) {
	min := s.min.LastFrame()
	max := s.max.LastFrame()
	trigger := s.trigger.LastFrame()
	size := s.size.LastFrame()
	clock := s.clock.LastFrame()

	for i := 0; i < times; i++ {
		if s.reads > 0 {
			op(i)
			continue
		}
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
		s.lastSize = size
		s.lastTrigger = trigger[i]
		s.lastClock = clock[i]

		op(i)
	}
}

type randomSeriesOut struct {
	*randomSeries
}

func (o *randomSeriesOut) Read(out Frame) {
	o.read()
	o.tick(len(out), func(i int) {
		out[i] = o.memory[o.idx]
	})
	o.postRead()
}

type randomSeriesGate struct {
	*randomSeries
}

func (o *randomSeriesGate) Read(out Frame) {
	o.read()
	clock := o.clock.LastFrame()
	o.tick(len(out), func(i int) {
		if o.gateMemory[o.idx] > 0 {
			out[i] = clock[i]
		} else {
			out[i] = -1
		}
	})
	o.postRead()
}
