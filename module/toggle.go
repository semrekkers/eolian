package module

func init() {
	Register("Toggle", func(Config) (Patcher, error) { return newToggle() })
}

type toggle struct {
	IO
	trigger            *In
	value, lastTrigger Value
}

func newToggle() (*toggle, error) {
	m := &toggle{
		trigger:     &In{Name: "trigger", Source: NewBuffer(zero)},
		lastTrigger: -1,
	}
	return m, m.Expose(
		"Toggle",
		[]*In{m.trigger},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
}

func (t *toggle) Read(out Frame) {
	trigger := t.trigger.ReadFrame()
	for i := range out {
		if t.lastTrigger <= 0 && trigger[i] > 0 {
			if t.value == 1 {
				t.value = -1
			} else {
				t.value = 1
			}
		}
		t.lastTrigger = trigger[i]
		out[i] = t.value
	}
}
