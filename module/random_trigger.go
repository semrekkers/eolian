package module

import "buddin.us/eolian/dsp"

func init() {
	Register("RandomTrigger", func(Config) (Patcher, error) { return newRandomTrigger() })
}

type randomTrigger struct {
	IO
	clock, probability *In
	lastClock          dsp.Float64
}

func newRandomTrigger() (*randomTrigger, error) {
	m := &randomTrigger{
		clock:       NewInBuffer("clock", dsp.Float64(-1)),
		probability: NewInBuffer("probability", dsp.Float64(1)),
		lastClock:   -1,
	}
	err := m.Expose(
		"RandomTrigger",
		[]*In{m.clock, m.probability},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (r *randomTrigger) Process(out dsp.Frame) {
	clock := r.clock.ProcessFrame()
	probability := r.probability.ProcessFrame()
	for i := range out {
		if dsp.Rand() <= probability[i] && r.lastClock < 0 && clock[i] > 0 {
			out[i] = 1
		} else {
			out[i] = -1
		}
	}
}
