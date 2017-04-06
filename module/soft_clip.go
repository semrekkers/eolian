package module

import (
	"buddin.us/eolian/dsp"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("SoftClip", func(c Config) (Patcher, error) {
		var config struct{}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		return newSoftClip()
	})
}

type softClip struct {
	IO
	in, gain *In
}

func newSoftClip() (*softClip, error) {
	m := &softClip{
		in:   NewIn("input", dsp.Float64(0)),
		gain: NewInBuffer("gain", dsp.Float64(1)),
	}
	return m, m.Expose(
		"SoftClip",
		[]*In{m.in, m.gain},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}})
}

func (m *softClip) Process(out dsp.Frame) {
	m.in.Process(out)
	gain := m.gain.ProcessFrame()
	for i := range out {
		abs := dsp.Abs(out[i])
		if abs <= 0.5 {
			continue
		}
		out[i] = ((abs - 0.25) / out[i]) * gain[i]
	}
}
