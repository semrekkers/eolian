package module

func init() {
	Register("Noise", func(Config) (Patcher, error) { return NewNoise() })
}

type Noise struct {
	IO
	in, min, max, gain *In
}

func NewNoise() (*Noise, error) {
	m := &Noise{
		in:   &In{Name: "input", Source: zero},
		min:  &In{Name: "min", Source: NewBuffer(zero)},
		max:  &In{Name: "max", Source: NewBuffer(Value(1))},
		gain: &In{Name: "gain", Source: NewBuffer(Value(1))},
	}
	err := m.Expose(
		[]*In{m.in, m.min, m.max, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Noise) Read(out Frame) {
	reader.in.Read(out)
	min := reader.min.ReadFrame()
	max := reader.max.ReadFrame()
	gain := reader.gain.ReadFrame()
	for i := range out {
		diff := max[i] - min[i]
		out[i] += (randValue()*diff + min[i]) * gain[i]
	}
}
