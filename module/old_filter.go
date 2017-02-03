package module

import (
	"fmt"
	"math"
)

func init() {
	Register("LPFilter", func(Config) (Patcher, error) { return newFilter(lowPass) })
	Register("HPFilter", func(Config) (Patcher, error) { return newFilter(highPass) })
}

type filterType int

const (
	lowPass filterType = iota
	highPass
	bandPass
)

type filter struct {
	IO
	in, cutoff, resonance *In
	fourPole              *fourPole
}

func newFilter(kind filterType) (*filter, error) {
	m := &filter{
		in:        &In{Name: "input", Source: zero},
		cutoff:    &In{Name: "cutoff", Source: NewBuffer(Frequency(1000))},
		resonance: &In{Name: "resonance", Source: NewBuffer(zero)},
		fourPole:  &fourPole{kind: kind},
	}

	var name string
	switch kind {
	case lowPass:
		name = "LPFilter"
	case highPass:
		name = "HPFilter"
	default:
		name = "UnknownFilter"
	}

	err := m.Expose(
		name,
		[]*In{m.in, m.cutoff, m.resonance},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (f *filter) Read(out Frame) {
	f.in.Read(out)
	cutoff := f.cutoff.ReadFrame()
	resonance := f.resonance.ReadFrame()
	for i := range out {
		f.fourPole.cutoff = cutoff[i]
		f.fourPole.resonance = resonance[i]
		out[i] = f.fourPole.Tick(out[i])
	}
}

type fourPole struct {
	kind      filterType
	cutoff    Value
	resonance Value
	after     [4]Value
}

func (filter *fourPole) Tick(in Value) Value {
	cutoff := Value(2 * math.Pi * math.Abs(float64(filter.cutoff)))
	fmt.Println(cutoff)

	res := filter.after[3]
	out := in - (res * clampValue(filter.resonance, 0, 4.5))

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
	case highPass:
		return out - filter.after[3]
	case lowPass:
		return filter.after[3]
	}

	return out
}
