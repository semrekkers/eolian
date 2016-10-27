package module

import "math"

func init() {
	Register("Compress", func(Config) (Patcher, error) { return NewCompress() })
}

type Compress struct {
	IO
	in, attack, release *In

	envelope Value
}

func NewCompress() (*Compress, error) {
	m := &Compress{
		in:      &In{Name: "input", Source: zero},
		attack:  &In{Name: "attack", Source: NewBuffer(Duration(10))},
		release: &In{Name: "release", Source: NewBuffer(Duration(500))},
	}
	err := m.Expose(
		[]*In{m.in, m.attack, m.release},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Compress) Read(out Frame) {
	reader.in.Read(out)
	attack, release := reader.attack.ReadFrame(), reader.release.ReadFrame()
	for i := range out {
		in := absValue(out[i])
		side := release[i]
		if in > reader.envelope {
			side = attack[i]
		}

		factor := math.Pow(0.01, float64(1.0/side))
		reader.envelope = Value(factor)*(reader.envelope-in) + in

		if reader.envelope > 1 {
			out[i] /= reader.envelope
		}
	}
}
