package module

import (
	"strconv"

	"github.com/mitchellh/mapstructure"
)

const (
	defaultReverbFeedback = Value(0.84)
	defaultReverbGain     = Value(0.5)
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
	in, feedback, fbCutoff, gain, bias, cutoff *In

	fbs       []*filteredFBDelay
	allpasses []*allpass
	filter    *svFilter
	inputs    *frInputs
}

func newFilteredReverb(c reverbConfig) (*filteredReverb, error) {
	feedbacks := len(c.Feedback)
	allpasses := len(c.Allpass)
	inputs, err := newFRInputs(feedbacks, allpasses)
	if err != nil {
		return nil, err
	}

	filter, err := newSVFilter(4)
	if err != nil {
		return nil, err
	}

	m := &filteredReverb{
		in:        &In{Name: "input", Source: inputs.input.in},
		feedback:  &In{Name: "feedback", Source: inputs.feedback.in},
		fbCutoff:  &In{Name: "fbCutoff", Source: inputs.fbCutoff.in},
		gain:      &In{Name: "gain", Source: inputs.gain.in},
		bias:      &In{Name: "bias", Source: inputs.crossfade.bias},
		cutoff:    &In{Name: "cutoff", Source: filter.cutoff},
		fbs:       make([]*filteredFBDelay, feedbacks),
		allpasses: make([]*allpass, allpasses),
		filter:    filter,
		inputs:    inputs,
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
	if err := m.setDefaults(); err != nil {
		return m, err
	}

	return m, m.Expose(
		"FilteredReverb",
		[]*In{m.in, m.feedback, m.fbCutoff, m.gain, m.bias, m.cutoff},
		[]*Out{{Name: "output", Provider: Provide(inputs.crossfade)}},
	)
}

func (m *filteredReverb) Reset() error {
	if err := m.IO.Reset(); err != nil {
		return err
	}
	return m.setDefaults()
}

func (m *filteredReverb) setDefaults() error {
	if err := m.filter.Patch("cutoff", Frequency(20000)); err != nil {
		return err
	}
	if err := m.inputs.feedback.Patch("input", defaultReverbFeedback); err != nil {
		return err
	}
	return m.inputs.gain.Patch("input", defaultReverbGain)
}

func (m *filteredReverb) patchWetDry(inputs *frInputs) error {
	inputs.crossfade.forcedActiveOutputs = 1

	dryOut, err := inputs.input.Output("0")
	if err != nil {
		return err
	}
	if err := inputs.crossfade.Patch("a", dryOut); err != nil {
		return err
	}

	wet, err := inputs.input.Output("1")
	if err != nil {
		return err
	}
	if err := inputs.wetIn.Patch("input", wet); err != nil {
		return err
	}
	wetOut, err := m.allpasses[len(m.allpasses)-1].Output("output")
	if err != nil {
		return err
	}
	if err := m.filter.Patch("input", wetOut); err != nil {
		return err
	}
	filteredOut, err := m.filter.Output("lowpass")
	if err != nil {
		return err
	}
	return inputs.crossfade.Patch("b", filteredOut)
}

func (m *filteredReverb) patchFeedbacks(inputs *frInputs, sizes []int) error {
	for i, s := range sizes {
		name := strconv.Itoa(i)
		var err error
		m.fbs[i], err = newFilteredFBDelay(DurationInt(s))
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
		if err := m.fbs[i].Patch("cutoff", Port{inputs.fbCutoff, name}); err != nil {
			return err
		}
		if err := inputs.mixer.Patch(name+".input", Port{m.fbs[i], "output"}); err != nil {
			return err
		}
	}
	return nil
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
	return nil
}

func (r *filteredReverb) LuaMembers() []string {
	return []string{
		r.inputs.gain.ID(),
		r.inputs.input.ID(),
		r.inputs.wetIn.ID(),
		r.inputs.crossfade.ID(),
		r.inputs.fbCutoff.ID(),
		r.inputs.feedback.ID(),
		r.filter.ID(),
	}
}

type frInputs struct {
	input, wetIn, feedback, fbCutoff, gain *multiple
	mixer                                  *mix
	crossfade                              *crossfade
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
	if modules.fbCutoff, err = newMultiple(feedbacks); err != nil {
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
