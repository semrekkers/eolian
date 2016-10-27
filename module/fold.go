package module

import "math"

func init() {
	Register("Fold", func(Config) (Patcher, error) { return NewFold() })
}

type Fold struct {
	IO
	in, level *In
}

func NewFold() (*Fold, error) {
	m := &Fold{
		in:    &In{Name: "input", Source: zero},
		level: &In{Name: "level", Source: NewBuffer(Value(1))},
	}
	err := m.Expose(
		[]*In{m.in, m.level},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Fold) Read(out Frame) {
	reader.in.Read(out)
	level := reader.level.ReadFrame()
	for i := range out {
		in := float64(out[i])
		level := math.Abs(float64(level[i]))
		if in > level || in < -level {
			out[i] = Value(math.Abs(math.Abs(math.Mod(in-level, level*4))-level*2) - level)
		}
	}
}
