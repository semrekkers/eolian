package module

func init() {
	Register("SampleHold", func(Config) (Patcher, error) { return NewSampleHold() })
}

type SampleHold struct {
	IO
	in, trigger *In

	sample, lastTrigger Value
}

func NewSampleHold() (*SampleHold, error) {
	m := &SampleHold{
		in:      &In{Name: "input", Source: zero},
		trigger: &In{Name: "trigger", Source: NewBuffer(zero)},
	}
	err := m.Expose(
		[]*In{m.in, m.trigger},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *SampleHold) Read(out Frame) {
	reader.in.Read(out)
	trigger := reader.trigger.ReadFrame()
	for i := range out {
		if reader.lastTrigger < 0 && trigger[i] > 0 {
			reader.sample = out[i]
		}
		out[i] = reader.sample
		reader.lastTrigger = trigger[i]
	}
}
