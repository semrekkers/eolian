package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Toggle", func(Config) (Patcher, error) { return newToggle() })
}

type toggle struct {
	IO
	trigger            *In
	value, lastTrigger dsp.Float64
}

func newToggle() (*toggle, error) {
	m := &toggle{
		trigger:     NewInBuffer("trigger", dsp.Float64(0)),
		lastTrigger: -1,
	}
	return m, m.Expose(
		"Toggle",
		[]*In{m.trigger},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
}

func (t *toggle) Process(out dsp.Frame) {
	trigger := t.trigger.ProcessFrame()
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
