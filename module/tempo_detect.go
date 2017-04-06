package module

import "buddin.us/eolian/dsp"

func init() {
	Register("TempoDetect", func(Config) (Patcher, error) { return newTempoDetect() })
}

type tempoDetect struct {
	IO
	tap *In

	tick             int
	capture, lastTap dsp.Float64
}

func newTempoDetect() (*tempoDetect, error) {
	m := &tempoDetect{
		tap: NewInBuffer("tap", dsp.Float64(0)),
	}
	return m, m.Expose(
		"TempoDetect",
		[]*In{m.tap},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
}

func (t *tempoDetect) Process(out dsp.Frame) {
	tap := t.tap.ProcessFrame()
	for i := range out {
		if t.lastTap < 0 && tap[i] > 0 {
			t.capture = dsp.Float64((dsp.SampleRate / float64(t.tick)) / dsp.SampleRate)
			t.tick = 0
		}
		out[i] = t.capture
		t.tick++
		t.lastTap = tap[i]
	}
}
