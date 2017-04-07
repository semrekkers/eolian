package module

import (
	"buddin.us/eolian/dsp"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Reverb", func(c Config) (Patcher, error) {
		var config struct {
			Feedback, Allpass []int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if len(config.Feedback) == 0 {
			config.Feedback = []int{4003, 3001, 2004, 1002, 3027}
		}
		if len(config.Allpass) == 0 {
			config.Allpass = []int{573, 331, 178}
		}
		return newReverb(config.Feedback, config.Allpass)
	})
}

type reverb struct {
	IO
	in, feedback, gain *In

	fb      []*dsp.FBComb
	allpass []*dsp.AllPass
}

func newReverb(feedback, allpass []int) (*reverb, error) {
	fbCount := len(feedback)
	apCount := len(allpass)

	m := &reverb{
		in:       NewIn("input", dsp.Float64(0)),
		feedback: NewInBuffer("feedback", dsp.Float64(0.84)),
		gain:     NewInBuffer("gain", dsp.Float64(0.5)),
		fb:       make([]*dsp.FBComb, fbCount),
		allpass:  make([]*dsp.AllPass, apCount),
	}

	for i, s := range feedback {
		m.fb[i] = dsp.NewFBComb(dsp.DurationInt(s))
	}
	for i, s := range allpass {
		m.allpass[i] = dsp.NewAllPass(dsp.DurationInt(s))
	}

	return m, m.Expose(
		"Reverb",
		[]*In{m.in, m.feedback, m.gain},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
}

func (m *reverb) Process(out dsp.Frame) {
	m.in.Process(out)
	gain := m.gain.ProcessFrame()
	feedback := m.feedback.ProcessFrame()

	for i := range out {
		var mix dsp.Float64
		for j := 0; j < len(m.fb); j++ {
			mix += m.fb[j].Tick(out[i], feedback[i])
		}
		for j := 0; j < len(m.allpass); j++ {
			mix = m.allpass[j].Tick(mix, gain[i])
		}
		out[i] = mix
	}
}
