package module

import "math"

func init() {
	Register("Fold", func(Config) (Patcher, error) { return newFold() })
}

type fold struct {
	IO
	in, level *In
}

func newFold() (*fold, error) {
	m := &fold{
		in:    &In{Name: "input", Source: zero},
		level: &In{Name: "level", Source: NewBuffer(Value(1))},
	}
	err := m.Expose(
		"Fold",
		[]*In{m.in, m.level},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (f *fold) Read(out Frame) {
	f.in.Read(out)
	level := f.level.ReadFrame()
	for i := range out {
		in := float64(out[i])
		level := math.Max(math.Abs(float64(level[i])), 0.00001)
		if in > level || in < -level {
			out[i] = Value(math.Abs(math.Abs(math.Mod(in-level, level*4))-level*2) - level)
		}
	}
}
