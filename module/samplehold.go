package module

func init() {
	Register("SampleHold", func(Config) (Patcher, error) { return newSampleHold() })
}

type sampleHold struct {
	IO
	in, trigger *In

	sample, lastTrigger Value
}

func newSampleHold() (*sampleHold, error) {
	m := &sampleHold{
		in:      &In{Name: "input", Source: zero},
		trigger: &In{Name: "trigger", Source: NewBuffer(zero)},
	}
	return m, m.Expose(
		"SampleHold",
		[]*In{m.in, m.trigger},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
}

func (sh *sampleHold) Read(out Frame) {
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
