package module

func init() {
	Register("Crossfade", func(Config) (Patcher, error) { return NewCrossfade() })
}

type Crossfade struct {
	IO
	a, b, bias *In
}

func NewCrossfade() (*Crossfade, error) {
	m := &Crossfade{
		a:    &In{Name: "a", Source: NewBuffer(zero)},
		b:    &In{Name: "b", Source: NewBuffer(zero)},
		bias: &In{Name: "bias", Source: NewBuffer(zero)},
	}
	err := m.Expose(
		[]*In{m.a, m.b, m.bias},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Crossfade) Read(out Frame) {
	a, b := reader.a.ReadFrame(), reader.b.ReadFrame()
	bias := reader.bias.ReadFrame()
	for i := range out {
		if bias[i] > 0 {
			out[i] = (1-bias[i])*a[i] + b[i]
		} else if bias[i] < 0 {
			out[i] = a[i] + (1-bias[i])*b[i]
		} else {
			out[i] = a[i] + b[i]
		}
	}
}