package module

import (
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
		in:   &In{Name: "input", Source: zero},
		gain: &In{Name: "gain", Source: NewBuffer(Value(1))},
	}
	return m, m.Expose(
		"SoftClip",
		[]*In{m.in, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}})
}

func (m *softClip) Read(out Frame) {
	m.in.Read(out)
	gain := m.gain.ReadFrame()
	for i := range out {
		abs := absValue(out[i])
		if abs <= 0.5 {
			continue
		}
		out[i] = ((abs - 0.25) / out[i]) * gain[i]
	}
}
