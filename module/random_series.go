package module

import (
	"math/rand"

	"buddin.us/eolian/dsp"
)

func init() {
	Register("RandomSeries", func(Config) (Patcher, error) { return newRandomSeries() })
}

const randomSeriesMax = 32

type randomSeries struct {
	multiOutIO
	clock, size, trigger, min, max *In
	idx                            int
	memory, gateMemory             []dsp.Float64
	lastTrigger, lastClock         dsp.Float64

	valueOut, gateOut dsp.Frame
}

func newRandomSeries() (*randomSeries, error) {
	m := &randomSeries{
		clock:       NewInBuffer("clock", dsp.Float64(0)),
		size:        NewInBuffer("size", dsp.Float64(8)),
		trigger:     NewInBuffer("trigger", dsp.Float64(0)),
		min:         NewInBuffer("min", dsp.Float64(0)),
		max:         NewInBuffer("max", dsp.Float64(1)),
		memory:      make([]dsp.Float64, randomSeriesMax),
		gateMemory:  make([]dsp.Float64, randomSeriesMax),
		valueOut:    dsp.NewFrame(),
		gateOut:     dsp.NewFrame(),
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

func (s *randomSeries) Process(out dsp.Frame) {
	s.incrRead(func() {
		var (
			min     = s.min.ProcessFrame()
			max     = s.max.ProcessFrame()
			trigger = s.trigger.ProcessFrame()
			size    = s.size.ProcessFrame()
			clock   = s.clock.ProcessFrame()
		)

		for i := range out {
			size := dsp.Clamp(size[i], 1, randomSeriesMax)
			if s.lastClock < 0 && clock[i] > 0 {
				s.idx++
				if s.idx >= int(size) {
					s.idx = 0
				}
			}
			if s.lastTrigger < 0 && trigger[i] > 0 {
				for j := 0; j < int(size); j++ {
					s.memory[j] = dsp.Rand()*(max[j]-min[j]) + min[j]
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
