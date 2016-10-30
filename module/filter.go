package module

import "math"

func init() {
	Register("LPFilter", func(Config) (Patcher, error) { return NewFilter(LowPass) })
	Register("HPFilter", func(Config) (Patcher, error) { return NewFilter(HighPass) })
}

const (
	LowPass int = iota
	HighPass
)

type Filter struct {
	IO
	in, cutoff, resonance *In
	fourPole              *FourPole
}

func NewFilter(kind int) (*Filter, error) {
	m := &Filter{
		in:        &In{Name: "input", Source: zero},
		cutoff:    &In{Name: "cutoff", Source: NewBuffer(zero)},
		resonance: &In{Name: "resonance", Source: NewBuffer(zero)},
		fourPole:  &FourPole{kind: kind},
	}
	err := m.Expose(
		[]*In{m.in, m.cutoff, m.resonance},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Filter) Read(out Frame) {
	reader.in.Read(out)
	cutoff := reader.cutoff.ReadFrame()
	resonance := reader.resonance.ReadFrame()
	for i := range out {
		reader.fourPole.cutoff = cutoff[i]
		reader.fourPole.resonance = resonance[i]
		out[i] = reader.fourPole.Tick(out[i])
	}
}

type FourPole struct {
	kind      int
	cutoff    Value
	resonance Value
	after     [4]Value
}

func (filter *FourPole) Tick(in Value) Value {
	cutoff := Value(2 * math.Pi * math.Abs(float64(filter.cutoff)))

	var out Value

	res := filter.after[3]
	out = in - (res * clampValue(filter.resonance, 0, 4.5))

	clip := out
	if clip > 1 {
		clip = 1
	} else if clip < -1 {
		clip = -1
	}

	out = clip + ((-clip + out) * 0.995)
	filter.after[0] += (-filter.after[0] + out) * cutoff
	filter.after[1] += (-filter.after[1] + filter.after[0]) * cutoff
	filter.after[2] += (-filter.after[2] + filter.after[1]) * cutoff
	filter.after[3] += (-filter.after[3] + filter.after[2]) * cutoff

	switch filter.kind {
	case HighPass:
		return out - filter.after[3]
	case LowPass:
		return filter.after[3]
	}

	return out
}
