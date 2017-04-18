package module

import (
	"math"

	"buddin.us/eolian/dsp"
)

func init() {
	Register("Fold", func(Config) (Patcher, error) { return newFold() })
}

type fold struct {
	IO
	in, stages, level, gain *In
}

func newFold() (*fold, error) {
	m := &fold{
		in:     NewIn("input", dsp.Float64(0)),
		stages: NewInBuffer("stages", dsp.Float64(1)),
		level:  NewInBuffer("level", dsp.Float64(1)),
		gain:   NewInBuffer("gain", dsp.Float64(1)),
	}
	err := m.Expose(
		"Fold",
		[]*In{m.in, m.stages, m.level, m.gain},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (f *fold) Process(out dsp.Frame) {
	f.in.Process(out)
	stages := f.stages.ProcessFrame()
	level := f.level.ProcessFrame()
	gain := f.gain.ProcessFrame()
	for i := range out {
		in := float64(out[i])
		level := math.Max(math.Abs(float64(level[i])), 0.00001)
		for j := 0; j < int(stages[i]); j++ {
			if in > level || in < -level {
				out[i] = dsp.Float64(math.Abs(math.Abs(math.Mod(in-level, level*4))-level*2) - level)
			}
			level *= 0.75
		}
		out[i] *= gain[i]
	}
}
