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
	feedbacks := len(c.Feedback)
	allpasses := len(c.Allpass)
	inputs, err := newFRInputs(feedbacks, allpasses)
	if err != nil {
		return nil, err
	}

	m := &filteredReverb{
		in:        &In{Name: "input", Source: inputs.input.in},
		feedback:  &In{Name: "feedback", Source: inputs.feedback.in},
		cutoff:    &In{Name: "cutoff", Source: inputs.cutoff.in},
		gain:      &In{Name: "gain", Source: inputs.gain.in},
		bias:      &In{Name: "bias", Source: inputs.crossfade.bias},
		fbs:       make([]*filteredFBComb, feedbacks),
		allpasses: make([]*allpass, allpasses),
	}

	if err := m.patchFeedbacks(inputs, c.Feedback); err != nil {
		return m, err
	}
	if err := m.patchAllpasses(inputs, c.Allpass); err != nil {
		return m, err
	}
	if err := m.patchWetDry(inputs); err != nil {
		return m, err
	}

	return m, m.Expose(
		"FilteredReverb",
		[]*In{m.in, m.feedback, m.cutoff, m.gain, m.bias},
		[]*Out{{Name: "output", Provider: Provide(inputs.crossfade)}},
	)
}

func (m *filteredReverb) patchWetDry(inputs *frInputs) error {
	wet, err := inputs.input.Output("1")
	if err != nil {
		return err
	}
	if err := inputs.wetIn.Patch("input", wet); err != nil {
		return err
	}
	dryOut, err := inputs.input.Output("0")
	if err != nil {
		return err
	}
	if err := inputs.crossfade.Patch("a", dryOut); err != nil {
		return err
	}
	wetOut, err := m.allpasses[len(m.allpasses)-1].Output("output")
	if err != nil {
		return err
	}
	return inputs.crossfade.Patch("b", wetOut)
}

func (m *filteredReverb) patchFeedbacks(inputs *frInputs, sizes []int) error {
	for i, s := range sizes {
		name := strconv.Itoa(i)
		var err error
		m.fbs[i], err = newFilteredFBComb(DurationInt(s))
		if err != nil {
			return err
		}
		if err := m.fbs[i].Patch("duration", DurationInt(s)); err != nil {
			return err
		}
		if err := m.fbs[i].Patch("input", Port{inputs.wetIn, name}); err != nil {
			return err
		}
		if err := m.fbs[i].Patch("gain", Port{inputs.feedback, name}); err != nil {
			return err
		}
		if err := m.fbs[i].Patch("cutoff", Port{inputs.cutoff, name}); err != nil {
			return err
		}
		if err := inputs.mixer.Patch(name+".input", Port{m.fbs[i], "output"}); err != nil {
			return err
		}
	}
	return inputs.feedback.Patch("input", Value(0.84))
}

func (m *filteredReverb) patchAllpasses(inputs *frInputs, sizes []int) error {
	for i, s := range sizes {
		var err error
		m.allpasses[i], err = newAllpass(DurationInt(s))
		if err != nil {
			return err
		}
		if err := m.allpasses[i].Patch("duration", DurationInt(s)); err != nil {
			return err
		}
		if i == 0 {
			if err := m.allpasses[i].Patch("input", Port{inputs.mixer, "output"}); err != nil {
				return err
			}
		} else {
			if err := m.allpasses[i].Patch("input", Port{m.allpasses[i-1], "output"}); err != nil {
				return err
			}
		}
		if err := m.allpasses[i].Patch("gain", Port{inputs.gain, strconv.Itoa(i)}); err != nil {
			return err
		}
	}
	return inputs.gain.Patch("input", Value(0.5))
}

type frInputs struct {
	input, wetIn, feedback, cutoff, gain *multiple
	mixer                                *mix
	crossfade                            *crossfade
}

func newFRInputs(feedbacks, allpasses int) (*frInputs, error) {
	var (
		modules = &frInputs{}
		err     error
	)

	if modules.input, err = newMultiple(2); err != nil {
		return nil, err
	}
	if modules.wetIn, err = newMultiple(feedbacks); err != nil {
		return nil, err
	}
	if modules.feedback, err = newMultiple(feedbacks); err != nil {
		return nil, err
	}
	if modules.cutoff, err = newMultiple(feedbacks); err != nil {
		return nil, err
	}
	if modules.gain, err = newMultiple(allpasses); err != nil {
		return nil, err
	}
	if modules.mixer, err = newMix(feedbacks); err != nil {
		return nil, err
	}
	if modules.crossfade, err = newCrossfade(); err != nil {
		return nil, err
	}
	return modules, nil
}
