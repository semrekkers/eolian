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

func (reader *Toggle) Read(out Frame) {
	trigger := reader.trigger.ReadFrame()
	for i := range out {
		if reader.lastTrigger < 0 && trigger[i] > 0 {
			if reader.value == 1 {
				reader.value = 0
			} else {
				reader.value = 1
			}
		}
		reader.lastTrigger = trigger[i]
		out[i] = reader.value
	}
}
