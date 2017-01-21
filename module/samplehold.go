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

func (sh *SampleHold) Read(out Frame) {
	sh.in.Read(out)
	trigger := sh.trigger.ReadFrame()
	for i := range out {
		if sh.lastTrigger < 0 && trigger[i] > 0 {
			sh.sample = out[i]
		}
		out[i] = sh.sample
		sh.lastTrigger = trigger[i]
	}
}
