package module

import "buddin.us/eolian/dsp"

func init() {
	Register("SampleHold", func(Config) (Patcher, error) { return newSampleHold() })
}

type sampleHold struct {
	IO
	in, trigger         *In
	sample, lastTrigger dsp.Float64
}

func newSampleHold() (*sampleHold, error) {
	m := &sampleHold{
		in:      NewIn("input", dsp.Float64(0)),
		trigger: NewInBuffer("trigger", dsp.Float64(0)),
	}
	return m, m.Expose(
		"SampleHold",
		[]*In{m.in, m.trigger},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
}

func (sh *sampleHold) Process(out dsp.Frame) {
	sh.in.Process(out)
	trigger := sh.trigger.ProcessFrame()
	for i := range out {
		if sh.lastTrigger < 0 && trigger[i] > 0 {
			sh.sample = out[i]
		}
		out[i] = sh.sample
		sh.lastTrigger = trigger[i]
	}
}
