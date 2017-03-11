package module

import (
	"fmt"

	lookup "github.com/brettbuddin/eolian/wavetable"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Wavetable", func(c Config) (Patcher, error) {
		var config struct{ Table string }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Table == "" {
			config.Table = "sine"
		}
		return newWavetable(config.Table)
	})
}

type wavetable struct {
	IO
	lut                *lookup.Table
	pitch, amp, offset *In
	lastPitch          Value
}

func newWavetable(tableName string) (*wavetable, error) {
	t, ok := lookup.Tables[tableName]
	if !ok {
		return nil, fmt.Errorf("unknown table %q", tableName)
	}

	m := &wavetable{
		pitch:  &In{Name: "pitch", Source: NewBuffer(zero)},
		amp:    &In{Name: "amp", Source: NewBuffer(Value(1))},
		offset: &In{Name: "offset", Source: NewBuffer(zero)},
		lut:    lookup.NewTable(t, len(t)/len(lookup.Breakpoints), SampleRate),
	}
	return m, m.Expose(
		"Wavetable",
		[]*In{m.pitch, m.amp, m.offset},
		[]*Out{{Name: "output", Provider: Provide(m)}})
}

func (t *wavetable) Read(out Frame) {
	pitch := t.pitch.ReadFrame()
	amp := t.amp.ReadFrame()
	offset := t.offset.ReadFrame()

	for i := range out {
		if t.lastPitch != pitch[i] {
			t.lut.SetDelta(float64(pitch[i]))
		}
		out[i] = Value(t.lut.Step())*amp[i] + offset[i]
		t.lastPitch = pitch[i]
	}
}
