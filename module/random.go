package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Random", func(Config) (Patcher, error) { return newRandom() })
}

type random struct {
	multiOutIO
	clock, smoothness, probability, min, max *In
	stepped, smooth                          dsp.Frame
	average                                  dsp.RollingAverage

	captured, lastClock dsp.Float64
}

func newRandom() (*random, error) {
	m := &random{
		clock:       NewInBuffer("clock", dsp.Float64(-1)),
		smoothness:  NewInBuffer("smoothness", dsp.Float64(1)),
		probability: NewInBuffer("probability", dsp.Float64(1)),
		min:         NewInBuffer("min", dsp.Float64(0)),
		max:         NewInBuffer("max", dsp.Float64(1)),
		stepped:     dsp.NewFrame(),
		smooth:      dsp.NewFrame(),
		lastClock:   -1,
	}
	err := m.Expose(
		"Random",
		[]*In{m.clock, m.smoothness, m.probability, m.min, m.max},
		[]*Out{
			{Name: "stepped", Provider: provideCopyOut(m, &m.stepped)},
			{Name: "smooth", Provider: provideCopyOut(m, &m.smooth)},
		},
	)
	return m, err
}

func (r *random) Process(out dsp.Frame) {
	r.incrRead(func() {
		clock := r.clock.ProcessFrame()
		smoothness := r.smoothness.ProcessFrame()
		probability := r.probability.ProcessFrame()
		min := r.min.ProcessFrame()
		max := r.max.ProcessFrame()

		for i := range out {
			if dsp.Rand() <= probability[i] && r.lastClock < 0 && clock[i] > 0 {
				r.captured = dsp.RandRange(min[i], max[i])
			}
			r.lastClock = clock[i]

			r.stepped[i] = r.captured
			r.average.Window = int(dsp.Max(1, smoothness[i]))
			r.smooth[i] = r.average.Tick(r.captured)
		}
	})
}
