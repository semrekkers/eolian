package module

import (
	"buddin.us/eolian/dsp"
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
	filter                            *dsp.SVFilter
}

func newLPG() (*lpg, error) {
	m := &lpg{
		in:        NewIn("input", dsp.Float64(0)),
		ctrl:      NewInBuffer("control", dsp.Float64(1)),
		mode:      NewInBuffer("mode", dsp.Float64(1)),
		cutoff:    NewInBuffer("cutoff", dsp.Frequency(20000)),
		resonance: NewInBuffer("resonance", dsp.Float64(1)),
		filter:    &dsp.SVFilter{Poles: 4},
	}
	return m, m.Expose("LPGate",
		[]*In{m.in, m.ctrl, m.mode, m.cutoff, m.resonance},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
}

func (l *lpg) Process(out dsp.Frame) {
	l.in.Process(out)

	var (
		mode   = l.mode.ProcessFrame()
		ctrl   = l.ctrl.ProcessFrame()
		cutoff = l.cutoff.ProcessFrame()
		res    = l.resonance.ProcessFrame()
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

func (l *lpg) applyFilter(in, ctrl, cutoff, res dsp.Float64) dsp.Float64 {
	l.filter.Cutoff = cutoff * ctrl
	l.filter.Resonance = res
	lp, _, _ := l.filter.Tick(in)
	return lp
}

const (
	lpgModeLowPass int = iota
	lpgModeCombo
	lpgModeAmplitude
)

func mapLPGMode(v dsp.Float64) int {
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
