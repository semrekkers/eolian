package module

func init() {
	Register("TempoDetect", func(Config) (Patcher, error) { return newTempoDetect() })
}

type tempoDetect struct {
	IO
	tap *In

	tick             int
	capture, lastTap Value
}

func newTempoDetect() (*tempoDetect, error) {
	m := &tempoDetect{
		tap: &In{Name: "tap", Source: NewBuffer(zero)},
	}
	return m, m.Expose(
		"TempoDetect",
		[]*In{m.tap},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
}

func (t *tempoDetect) Read(out Frame) {
	tap := t.tap.ReadFrame()
	for i := range out {
		if t.lastTap < 0 && tap[i] > 0 {
			t.capture = Value((SampleRate / float64(t.tick)) / SampleRate)
			t.tick = 0
		}
		out[i] = t.capture
		t.tick++
		t.lastTap = tap[i]
	}
}
