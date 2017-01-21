package module

import "math"

func init() {
	Register("Wrap", func(Config) (Patcher, error) { return NewWrap() })
}

type Wrap struct {
	IO
	in, level *In
}

func NewWrap() (*Wrap, error) {
	m := &Wrap{
		in:    &In{Name: "input", Source: zero},
		level: &In{Name: "level", Source: NewBuffer(Value(1))},
	}
	err := m.Expose(
		[]*In{m.in, m.level},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (w *Wrap) Read(out Frame) {
	w.in.Read(out)
	level := w.level.ReadFrame()
	for i := range out {
		in := float64(out[i])
		level := math.Abs(float64(level[i]))
		if in > level {
			out[i] = Value(in - 2*level)
		} else if in < -level {
			out[i] = Value(in + 2*level)
		}
	}
}
