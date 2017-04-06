package module

import (
	"fmt"

	"buddin.us/eolian/dsp"
	lookup "buddin.us/eolian/wavetable"
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
	lastPitch          dsp.Float64
}

func newWavetable(tableName string) (*wavetable, error) {
	t, ok := lookup.Tables[tableName]
	if !ok {
		return nil, fmt.Errorf("unknown table %q", tableName)
	}

	m := &wavetable{
		pitch:  NewInBuffer("pitch", dsp.Float64(0)),
		amp:    NewInBuffer("amp", dsp.Float64(1)),
		offset: NewInBuffer("offset", dsp.Float64(0)),
		lut:    lookup.NewTable(t, len(t)/len(lookup.Breakpoints), dsp.SampleRate),
	}
	return m, m.Expose(
		"Wavetable",
		[]*In{m.pitch, m.amp, m.offset},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}})
}

func (t *wavetable) Process(out dsp.Frame) {
	pitch := t.pitch.ProcessFrame()
	amp := t.amp.ProcessFrame()
	offset := t.offset.ProcessFrame()

	for i := range out {
		if t.lastPitch != pitch[i] {
			t.lut.SetDelta(float64(pitch[i]))
		}
		out[i] = dsp.Float64(t.lut.Step())*amp[i] + offset[i]
		t.lastPitch = pitch[i]
	}
}
