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
	in, level *In
}

func newFold() (*fold, error) {
	m := &fold{
		in:    NewIn("input", dsp.Float64(0)),
		level: NewInBuffer("level", dsp.Float64(1)),
	}
	err := m.Expose(
		"Fold",
		[]*In{m.in, m.level},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (f *fold) Process(out dsp.Frame) {
	f.in.Process(out)
	level := f.level.ProcessFrame()
	for i := range out {
		in := float64(out[i])
		level := math.Max(math.Abs(float64(level[i])), 0.00001)
		if in > level || in < -level {
			out[i] = dsp.Float64(math.Abs(math.Abs(math.Mod(in-level, level*4))-level*2) - level)
		}
	}
}
