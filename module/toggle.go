package module

func init() {
	Register("Toggle", func(Config) (Patcher, error) { return NewToggle() })
}

type Toggle struct {
	IO
	trigger            *In
	value, lastTrigger Value
}

func NewToggle() (*Toggle, error) {
	m := &Toggle{
		trigger: &In{Name: "trigger", Source: NewBuffer(zero)},
	}
	err := m.Expose(
		[]*In{m.trigger},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (t *Toggle) Read(out Frame) {
	trigger := t.trigger.ReadFrame()
	for i := range out {
		if t.lastTrigger < 0 && trigger[i] > 0 {
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
