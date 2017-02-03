package module

import (
	"strings"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Filter", func(c Config) (Patcher, error) {
		var config struct {
			Mode  string
			Poles int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}

		var mode filterType
		switch strings.ToLower(config.Mode) {
		case "low":
			fallthrough
		case "lowpass":
			mode = lowPass
		case "high":
			fallthrough
		case "highpass":
			mode = highPass
		case "band":
			fallthrough
		case "bandpass":
			mode = bandPass
		default:
			mode = lowPass
		}

		if config.Poles == 0 {
			config.Poles = 4
		}

		return newSVFilter(mode, config.Poles)
	})
}

type svFilter struct {
	IO
	in, cutoff, resonance         *In
	mode                          filterType
	poles                         int
	lastCutoff, g, state1, state2 Value
}

func newSVFilter(mode filterType, poles int) (*svFilter, error) {
	m := &svFilter{
		in:        &In{Name: "input", Source: zero},
		cutoff:    &In{Name: "cutoff", Source: NewBuffer(Frequency(1000))},
		resonance: &In{Name: "resonance", Source: NewBuffer(Value(1))},
		mode:      mode,
		poles:     poles,
	}

	err := m.Expose(
		"Filter",
		[]*In{m.in, m.cutoff, m.resonance},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (f *svFilter) Read(out Frame) {
	f.in.Read(out)
	cutoff := f.cutoff.ReadFrame()
	resonance := f.resonance.ReadFrame()

	for i := range out {
		if cutoff[i] != f.lastCutoff {
			f.g = tanValue(cutoff[i])
		}
		r := Value(1 / resonance[i])
		h := 1 / (1 + r*f.g + f.g*f.g)

		for j := 0; j < f.poles; j++ {
			hp := h * (out[i] - r*f.state1 - f.g*f.state1 - f.state2)
			bp := f.g*hp + f.state1
			lp := f.g*bp + f.state2

			f.state1 = f.g*hp + bp
			f.state2 = f.g*bp + lp

			switch f.mode {
			case lowPass:
				out[i] = lp
			case highPass:
				out[i] = hp
			case bandPass:
				out[i] = r * bp
			}
		}
	}
}
