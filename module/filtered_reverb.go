package module

import (
	"strconv"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("FilteredReverb", func(c Config) (Patcher, error) {
		var config reverbConfig
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if len(config.Feedback) == 0 {
			config.Feedback = []int{1557, 1617, 1491, 1422, 1277, 1356, 1118, 1116}
		}
		if len(config.Allpass) == 0 {
			config.Allpass = []int{225, 556, 441, 341}
		}
		return newFilteredReverb(config)
	})
}

type filteredReverb struct {
	IO
	in, feedback, cutoff, gain, bias *In

	fbs       []*filteredFBComb
	allpasses []*allpass
}

func newFilteredReverb(c reverbConfig) (*filteredReverb, error) {
	feedbackCount := len(c.Feedback)
	allpassCount := len(c.Allpass)

	input, err := newMultiple(2)
	if err != nil {
		return nil, err
	}
	wetIn, err := newMultiple(feedbackCount)
	if err != nil {
		return nil, err
	}
	feedback, err := newMultiple(feedbackCount)
	if err != nil {
		return nil, err
	}
	cutoff, err := newMultiple(feedbackCount)
	if err != nil {
		return nil, err
	}
	gain, err := newMultiple(allpassCount)
	if err != nil {
		return nil, err
	}
	mixer, err := newMix(feedbackCount)
	if err != nil {
		return nil, err
	}
	crossfade, err := newCrossfade()
	if err != nil {
		return nil, err
	}

	m := &filteredReverb{
		in:       &In{Name: "input", Source: input.in},
		feedback: &In{Name: "feedback", Source: feedback.in},
		cutoff:   &In{Name: "cutoff", Source: cutoff.in},
		gain:     &In{Name: "gain", Source: gain.in},
		bias:     &In{Name: "bias", Source: crossfade.bias},

		fbs:       make([]*filteredFBComb, feedbackCount),
		allpasses: make([]*allpass, allpassCount),
	}

	wet, err := input.Output("1")
	if err != nil {
		return m, err
	}
	if err := wetIn.Patch("input", wet); err != nil {
		return m, err
	}

	for i, s := range c.Feedback {
		name := strconv.Itoa(i)
		m.fbs[i], err = newFilteredFBComb(DurationInt(s))
		if err != nil {
			return m, err
		}
		if err := m.fbs[i].Patch("duration", DurationInt(s)); err != nil {
			return m, err
		}
		if err := m.fbs[i].Patch("input", Port{wetIn, name}); err != nil {
			return m, err
		}
		if err := m.fbs[i].Patch("gain", Port{feedback, name}); err != nil {
			return m, err
		}
		if err := m.fbs[i].Patch("cutoff", Port{cutoff, name}); err != nil {
			return m, err
		}
		if err := mixer.Patch(name+".input", Port{m.fbs[i], "output"}); err != nil {
			return m, err
		}
	}
	feedback.Patch("input", Value(0.84))

	for i, s := range c.Allpass {
		m.allpasses[i], err = newAllpass(DurationInt(s))
		if err != nil {
			return m, err
		}
		if err := m.allpasses[i].Patch("duration", DurationInt(s)); err != nil {
			return m, err
		}
		if i == 0 {
			if err := m.allpasses[i].Patch("input", Port{mixer, "output"}); err != nil {
				return m, err
			}
		} else {
			if err := m.allpasses[i].Patch("input", Port{m.allpasses[i-1], "output"}); err != nil {
				return m, err
			}
		}
		if err := m.allpasses[i].Patch("gain", Port{gain, strconv.Itoa(i)}); err != nil {
			return m, err
		}
	}
	gain.Patch("input", Value(0.5))

	dryOut, err := input.Output("0")
	if err != nil {
		return m, err
	}
	if err := crossfade.Patch("a", dryOut); err != nil {
		return m, err
	}

	wetOut, err := m.allpasses[len(m.allpasses)-1].Output("output")
	if err != nil {
		return m, err
	}
	if err := crossfade.Patch("b", wetOut); err != nil {
		return m, err
	}

	err = m.Expose(
		"FilteredReverb",
		[]*In{m.in, m.feedback, m.cutoff, m.gain, m.bias},
		[]*Out{{Name: "output", Provider: Provide(crossfade)}},
	)
	return m, err
}
