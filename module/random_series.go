package module

import "math/rand"

func init() {
	Register("RandomSeries", func(Config) (Patcher, error) { return NewRandomSeries() })
}

type RandomSeries struct {
	IO
	clock, size, trigger, rest *In
	min, max                   *In

	lastClock, lastTrigger Value
	memory                 []Value
	idx                    int
}

func NewRandomSeries() (*RandomSeries, error) {
	m := &RandomSeries{
		clock:       &In{Name: "clock", Source: NewBuffer(zero)},
		size:        &In{Name: "size", Source: NewBuffer(Value(8))},
		trigger:     &In{Name: "trigger", Source: NewBuffer(zero)},
		min:         &In{Name: "min", Source: NewBuffer(zero)},
		max:         &In{Name: "max", Source: NewBuffer(Value(1))},
		rest:        &In{Name: "rest", Source: NewBuffer(Value(1))},
		memory:      make([]Value, 32),
		lastTrigger: -1,
		lastClock:   -1,
	}
	err := m.Expose(
		[]*In{m.size, m.trigger, m.clock, m.min, m.max},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *RandomSeries) Read(out Frame) {
	size := reader.size.ReadFrame()
	trigger := reader.trigger.ReadFrame()
	clock := reader.clock.ReadFrame()
	min := reader.min.ReadFrame()
	max := reader.max.ReadFrame()
	rest := reader.rest.ReadFrame()

	for i := range out {
		size := clampValue(size[i], 1, 32)
		if reader.lastTrigger < 0 && trigger[i] > 0 {
			for i := 0; i < int(size); i++ {
				if rest[i] == 0 || (rest[i] == 1 && rand.Float32() > 0.25) {
					reader.memory[i] = randValue()*(max[i]-min[i]) + min[i]
				} else {
					reader.memory[i] = 0
				}
			}
		}
		out[i] = reader.memory[reader.idx]
		if reader.lastClock < 0 && clock[i] > 0 {
			reader.idx++
			if reader.idx >= int(size) {
				reader.idx = 0
			}
		}
		reader.lastTrigger = trigger[i]
		reader.lastClock = clock[i]
	}
}
