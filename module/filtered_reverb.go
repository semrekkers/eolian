package module

import (
	"buddin.us/eolian/dsp"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("FilteredReverb", func(c Config) (Patcher, error) {
		var config struct {
			Feedback, Allpass []int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if len(config.Feedback) == 0 {
			config.Feedback = []int{1557, 1617, 1491, 1422, 1277, 1356, 1118, 1116}
		}
		if len(config.Allpass) == 0 {
			config.Allpass = []int{225, 556, 441, 341}
		}
		return newFilteredReverb(config.Feedback, config.Allpass)
	})
}

type filteredReverb struct {
	IO
	in, feedback, fbCutoff, gain, bias, cutoff *In

	fb      []*dsp.FilteredFBComb
	allpass []*dsp.AllPass
	filter  *dsp.SVFilter
}

func newFilteredReverb(feedback, allpass []int) (*filteredReverb, error) {
	fbCount := len(feedback)
	apCount := len(allpass)

	m := &filteredReverb{
		in:       NewIn("input", dsp.Float64(0)),
		feedback: NewInBuffer("feedback", dsp.Float64(0.84)),
		fbCutoff: NewInBuffer("fbCutoff", dsp.Frequency(1000)),
		gain:     NewInBuffer("gain", dsp.Float64(0.5)),
		cutoff:   NewInBuffer("cutoff", dsp.Frequency(1000)),
		bias:     NewInBuffer("bias", dsp.Float64(0)),
		fb:       make([]*dsp.FilteredFBComb, fbCount),
		allpass:  make([]*dsp.AllPass, apCount),
		filter:   &dsp.SVFilter{Poles: 4, Resonance: 1},
	}

	for i, s := range feedback {
		m.fb[i] = dsp.NewFilteredFBComb(dsp.DurationInt(s), 4)
	}
	for i, s := range allpass {
		m.allpass[i] = dsp.NewAllPass(dsp.DurationInt(s))
	}

	return m, m.Expose(
		"FilteredReverb",
		[]*In{m.in, m.feedback, m.fbCutoff, m.gain, m.bias, m.cutoff},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
}

func (m *filteredReverb) Process(out dsp.Frame) {
	m.in.Process(out)
	gain := m.gain.ProcessFrame()
	feedback := m.feedback.ProcessFrame()
	fbCutoff := m.fbCutoff.ProcessFrame()
	cutoff := m.cutoff.ProcessFrame()
	bias := m.bias.ProcessFrame()

	for i := range out {
		dry := out[i]
		var wet dsp.Float64
		for _, fb := range m.fb {
			wet += fb.Tick(out[i], feedback[i], fbCutoff[i], 1)
		}
		for _, ap := range m.allpass {
			wet = ap.Tick(wet, gain[i])
		}
		m.filter.Cutoff = cutoff[i]
		wet, _, _ = m.filter.Tick(wet)
		out[i] = dsp.AttenSum(bias[i], dry, wet)
	}
}
