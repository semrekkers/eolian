package module

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Reverb", func(c Config) (Patcher, error) {
		var config reverbConfig
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if len(config.Feedback) == 0 {
			config.Feedback = []int{4003, 3001, 2004, 1002, 3027}
		}
		if len(config.Allpass) == 0 {
			config.Allpass = []int{573, 331, 178}
		}
		return newReverb(config)
	})
}

type reverbConfig struct {
	Feedback, Allpass []int
}

type reverb struct {
	IO
	in, feedback, gain *In

	fbs       []*fbComb
	allpasses []*allpass
}

func newReverb(c reverbConfig) (*reverb, error) {
	feedbackCount := len(c.Feedback)
	allpassCount := len(c.Allpass)

	inMultiple, err := newMultiple(feedbackCount)
	if err != nil {
		return nil, err
	}
	feedbackGainMultiple, err := newMultiple(feedbackCount)
	if err != nil {
		return nil, err
	}
	allpassGainMultiple, err := newMultiple(allpassCount)
	if err != nil {
		return nil, err
	}

	m := &reverb{
		in:       &In{Name: "input", Source: inMultiple.in},
		feedback: &In{Name: "feedback", Source: feedbackGainMultiple.in},
		gain:     &In{Name: "gain", Source: allpassGainMultiple.in},

		fbs:       make([]*fbComb, feedbackCount),
		allpasses: make([]*allpass, allpassCount),
	}

	mixer, err := newMix(feedbackCount)
	if err != nil {
		return m, err
	}
	for i, s := range c.Feedback {
		name := fmt.Sprintf("%d", i)
		m.fbs[i], err = newFBComb(DurationInt(s))
		if err != nil {
			return m, err
		}
		if err := m.fbs[i].Patch("duration", DurationInt(s)); err != nil {
			return m, err
		}
		if err := m.fbs[i].Patch("input", Port{inMultiple, name}); err != nil {
			return m, err
		}
		if err := m.fbs[i].Patch("gain", Port{feedbackGainMultiple, name}); err != nil {
			return m, err
		}
		if err := mixer.Patch(name+".input", Port{m.fbs[i], "output"}); err != nil {
			return m, err
		}
	}
	feedbackGainMultiple.Patch("input", Value(0.8))

	for i, s := range c.Allpass {
		m.allpasses[i], err = newAllpass(DurationInt(s))
		if err != nil {
			return m, err
		}
		if err := m.allpasses[i].Patch("duration", DurationInt(s)); err != nil {
			return m, err
		}
		var port Port
		if i == 0 {
			port = Port{mixer, "output"}
		} else {
			port = Port{m.allpasses[i-1], "output"}
		}
		if err := m.allpasses[i].Patch("input", port); err != nil {
			return m, err
		}
		if err := m.allpasses[i].Patch("gain", Port{allpassGainMultiple, fmt.Sprintf("%d", i)}); err != nil {
			return m, err
		}
	}
	allpassGainMultiple.Patch("input", Value(0.7))

	return m, m.Expose(
		"Reverb",
		[]*In{m.in, m.feedback, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m.allpasses[len(m.allpasses)-1])}},
	)
}
