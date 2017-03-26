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

	fbs       []*fbDelay
	allpasses []*allpass
	inputs    *rInputs
}

func newReverb(c reverbConfig) (*reverb, error) {
	feedbacks := len(c.Feedback)
	allpasses := len(c.Allpass)
	inputs, err := newRInputs(feedbacks, allpasses)
	if err != nil {
		return nil, err
	}

	m := &reverb{
		in:        &In{Name: "input", Source: inputs.input.in},
		feedback:  &In{Name: "feedback", Source: inputs.feedback.in},
		gain:      &In{Name: "gain", Source: inputs.gain.in},
		fbs:       make([]*fbDelay, feedbacks),
		allpasses: make([]*allpass, allpasses),
		inputs:    inputs,
	}

	mixer, err := newMix(feedbacks)
	if err != nil {
		return m, err
	}
	if err := m.patchFeedbacks(mixer, inputs, c.Feedback); err != nil {
		return m, err
	}
	if err := m.patchAllpasses(mixer, inputs, c.Allpass); err != nil {
		return m, err
	}
	if err := m.setDefaults(); err != nil {
		return m, err
	}

	m.allpasses[len(m.allpasses)-1].forcedActiveOutputs = 1

	return m, m.Expose(
		"Reverb",
		[]*In{m.in, m.feedback, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m.allpasses[len(m.allpasses)-1])}},
	)
}

func (m *reverb) Reset() error {
	if err := m.IO.Reset(); err != nil {
		return err
	}
	return m.setDefaults()
}

func (m *reverb) setDefaults() error {
	if err := m.inputs.feedback.Patch("input", defaultReverbFeedback); err != nil {
		return err
	}
	return m.inputs.gain.Patch("input", defaultReverbGain)
}

func (m *reverb) patchFeedbacks(mixer *mix, inputs *rInputs, sizes []int) error {
	for i, s := range sizes {
		name := fmt.Sprintf("%d", i)
		var err error
		m.fbs[i], err = newFBDelay(DurationInt(s))
		if err != nil {
			return err
		}
		if err := m.fbs[i].Patch("duration", DurationInt(s)); err != nil {
			return err
		}
		if err := m.fbs[i].Patch("input", Port{inputs.input, name}); err != nil {
			return err
		}
		if err := m.fbs[i].Patch("gain", Port{inputs.feedback, name}); err != nil {
			return err
		}
		if err := mixer.Patch(name+".input", Port{m.fbs[i], "output"}); err != nil {
			return err
		}
	}
	return nil
}

func (m *reverb) patchAllpasses(mixer *mix, inputs *rInputs, sizes []int) error {
	for i, s := range sizes {
		var err error
		m.allpasses[i], err = newAllpass(DurationInt(s))
		if err != nil {
			return err
		}
		if err := m.allpasses[i].Patch("duration", DurationInt(s)); err != nil {
			return err
		}
		var port Port
		if i == 0 {
			port = Port{mixer, "output"}
		} else {
			port = Port{m.allpasses[i-1], "output"}
		}
		if err := m.allpasses[i].Patch("input", port); err != nil {
			return err
		}
		if err := m.allpasses[i].Patch("gain", Port{inputs.gain, fmt.Sprintf("%d", i)}); err != nil {
			return err
		}
	}
	return nil
}

func (r *reverb) LuaMembers() []string {
	return []string{
		r.inputs.feedback.ID(),
		r.inputs.gain.ID(),
		r.inputs.input.ID(),
	}
}

type rInputs struct {
	input, feedback, gain *multiple
}

func newRInputs(feedbacks, allpasses int) (*rInputs, error) {
	var (
		modules = &rInputs{}
		err     error
	)

	if modules.input, err = newMultiple(feedbacks); err != nil {
		return nil, err
	}
	if modules.feedback, err = newMultiple(feedbacks); err != nil {
		return nil, err
	}
	if modules.gain, err = newMultiple(allpasses); err != nil {
		return nil, err
	}
	return modules, nil
}
