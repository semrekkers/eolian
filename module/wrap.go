package module

import (
	"math"

	"buddin.us/eolian/dsp"
)

func init() {
	Register("Wrap", func(Config) (Patcher, error) { return newWrap() })
}

type wrap struct {
	IO
	in, level *In
}

func newWrap() (*wrap, error) {
	m := &wrap{
		in:    NewIn("input", dsp.Float64(0)),
		level: NewInBuffer("level", dsp.Float64(1)),
	}
	err := m.Expose(
		"Wrap",
		[]*In{m.in, m.level},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (w *wrap) Process(out dsp.Frame) {
	w.in.Process(out)
	level := w.level.ProcessFrame()
	for i := range out {
		in := float64(out[i])
		level := math.Abs(float64(level[i]))
		if in > level {
			out[i] = dsp.Float64(in - 2*level)
		} else if in < -level {
			out[i] = dsp.Float64(in + 2*level)
		}
	}
}
