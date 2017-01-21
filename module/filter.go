package module

import "math"

func init() {
	Register("LPFilter", func(Config) (Patcher, error) { return NewFilter(LowPass) })
	Register("HPFilter", func(Config) (Patcher, error) { return NewFilter(HighPass) })
}

type FilterType int

const (
	LowPass FilterType = iota
	HighPass
)

type Filter struct {
	IO
	in, cutoff, resonance *In
	fourPole              *FourPole
}

func NewFilter(kind FilterType) (*Filter, error) {
	m := &Filter{
		in:        &In{Name: "input", Source: zero},
		cutoff:    &In{Name: "cutoff", Source: NewBuffer(Frequency(1000))},
		resonance: &In{Name: "resonance", Source: NewBuffer(zero)},
		fourPole:  &FourPole{kind: kind},
	}
	err := m.Expose(
		[]*In{m.in, m.cutoff, m.resonance},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (f *Filter) Read(out Frame) {
	f.in.Read(out)
	cutoff := f.cutoff.ReadFrame()
	resonance := f.resonance.ReadFrame()
	for i := range out {
		f.fourPole.cutoff = cutoff[i]
		f.fourPole.resonance = resonance[i]
		out[i] = f.fourPole.Tick(out[i])
	}
}

type FourPole struct {
	kind      FilterType
	cutoff    Value
	resonance Value
	after     [4]Value
}

func (filter *FourPole) Tick(in Value) Value {
	cutoff := Value(2 * math.Pi * math.Abs(float64(filter.cutoff)))

	var out Value

	var res Value
	if filter.resonance > 0 {
		res = filter.after[3]
	}
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
