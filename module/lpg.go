package module

import (
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("LPGate", func(c Config) (Patcher, error) {
		var config struct{}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		return newLPG()
	})
}

type lpg struct {
	IO
	in, ctrl, mode, cutoff, resonance *In
	filter                            *filter
}

func newLPG() (*lpg, error) {
	m := &lpg{
		in:        &In{Name: "input", Source: zero},
		ctrl:      &In{Name: "control", Source: NewBuffer(Value(1))},
		mode:      &In{Name: "mode", Source: NewBuffer(Value(1))},
		cutoff:    &In{Name: "cutoff", Source: NewBuffer(Frequency(20000))},
		resonance: &In{Name: "resonance", Source: NewBuffer(Value(1))},
		filter:    &filter{poles: 4},
	}
	return m, m.Expose("LPGate",
		[]*In{m.in, m.ctrl, m.mode, m.cutoff, m.resonance},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
}

func (l *lpg) Read(out Frame) {
	l.in.Read(out)

	var (
		mode   = l.mode.ReadFrame()
		ctrl   = l.ctrl.ReadFrame()
		cutoff = l.cutoff.ReadFrame()
		res    = l.resonance.ReadFrame()
	)

	for i := range out {
		switch mapLPGMode(mode[i]) {
		case lpgModeLowPass:
			out[i] = l.applyFilter(out[i], ctrl[i], cutoff[i], res[i])
		case lpgModeCombo:
			out[i] = l.applyFilter(out[i], ctrl[i], cutoff[i], res[i])
			out[i] *= ctrl[i]
		case lpgModeAmplitude:
			out[i] *= ctrl[i]
		}
	}
}

func (l *lpg) applyFilter(in, ctrl, cutoff, res Value) Value {
	l.filter.cutoff = cutoff * ctrl
	l.filter.resonance = res
	lp, _, _ := l.filter.Tick(in)
	return lp
}

const (
	lpgModeLowPass int = iota
	lpgModeCombo
	lpgModeAmplitude
)

func mapLPGMode(v Value) int {
	switch int(v) {
	case 0:
		return lpgModeLowPass
	case 1:
		return lpgModeCombo
	case 2:
		return lpgModeAmplitude
	default:
		return lpgModeCombo
	}
}
