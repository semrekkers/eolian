package module

func init() {
	Register("TempoDetect", func(Config) (Patcher, error) { return NewTempoDetect() })
}

type TempoDetect struct {
	IO
	tap *In

	tick             int
	capture, lastTap Value
}

func NewTempoDetect() (*TempoDetect, error) {
	m := &TempoDetect{
		tap: &In{Name: "tap", Source: NewBuffer(zero)},
	}
	err := m.Expose(
		[]*In{m.tap},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *TempoDetect) Read(out Frame) {
	tap := reader.tap.ReadFrame()
	for i := range out {
		if reader.lastTap < 0 && tap[i] > 0 {
			reader.capture = Value((SampleRate / float64(reader.tick)) / SampleRate)
			reader.tick = 0
		}
		out[i] = reader.capture
		reader.tick++
		reader.lastTap = tap[i]
	}
}
