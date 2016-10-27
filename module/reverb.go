package module

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

const inFeedback = "feedback"

func init() {
	Register("Reverb", func(c Config) (Patcher, error) {
		var config ReverbConfig
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if len(config.Feedback) == 0 {
			config.Feedback = []int{23, 29, 31, 41}
		}
		if len(config.Allpass) == 0 {
			config.Allpass = []int{83, 53, 23}
		}
		return NewReverb(config)
	})
}

type ReverbConfig struct {
	Feedback, Allpass []int
}

type Reverb struct {
	IO
	in, feedback, gain *In

	fbs       []*FBComb
	allpasses []*AllPass
}

func NewReverb(c ReverbConfig) (*Reverb, error) {
	feedbackCount := len(c.Feedback)
	allpassCount := len(c.Allpass)

	inMultiple, err := NewMultipleSized(feedbackCount)
	if err != nil {
		return nil, err
	}
	feedbackGainMultiple, err := NewMultipleSized(feedbackCount)
	if err != nil {
		return nil, err
	}
	allpassGainMultiple, err := NewMultipleSized(allpassCount)
	if err != nil {
		return nil, err
	}

	m := &Reverb{
		in:       &In{Name: "input", Source: inMultiple.in},
		feedback: &In{Name: "feedback", Source: feedbackGainMultiple.in},
		gain:     &In{Name: "gain", Source: allpassGainMultiple.in},

		fbs:       make([]*FBComb, feedbackCount),
		allpasses: make([]*AllPass, allpassCount),
	}

	mixer, err := NewMixSized(feedbackCount)
	if err != nil {
		return m, err
	}
	for i, s := range c.Feedback {
		name := fmt.Sprintf("%d", i)
		m.fbs[i], err = NewFBComb(s)
		if err != nil {
			return m, err
		}
		if err := m.fbs[i].Patch("duration", Value(1)); err != nil {
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
		m.allpasses[i], err = NewAllPass(s)
		if err != nil {
			return m, err
		}
		if err := m.allpasses[i].Patch("duration", Value(1)); err != nil {
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
		if err := m.allpasses[i].Patch("gain", Port{allpassGainMultiple, fmt.Sprintf("%d", i)}); err != nil {
			return m, err
		}
	}
	allpassGainMultiple.Patch("input", Value(0.7))

	err = m.Expose(
		[]*In{m.in, m.feedback, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m.allpasses[len(m.allpasses)-1])}},
	)
	return m, err
}
