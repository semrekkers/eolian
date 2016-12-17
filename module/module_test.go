package module

import (
	"strings"
	"testing"

	"gopkg.in/go-playground/assert.v1"
)

var defaultOutput = []string{"output"}

func TestRegisteredModules(t *testing.T) {
	modules := []struct {
		Name            string
		Config          Config
		Inputs, Outputs []string
	}{
		{"ADSR", nil, []string{
			"attack",
			"decay",
			"sustain",
			"release",
			"disableSustain",
			"ratio",
		}, defaultOutput},
		{"AND", nil, []string{"a", "b"}, defaultOutput},
		{"Allpass", nil, []string{"input", "duration", "gain"}, defaultOutput},
		{"Clip", nil, []string{"input", "max"}, defaultOutput},
		{"ClockMultiply", nil, []string{"input", "multiplier"}, defaultOutput},
		{"ClockDivide", nil, []string{"input", "divisor"}, defaultOutput},
		{"Compress", nil, []string{"input", "attack", "release"}, defaultOutput},
		{"Crossfade", nil, []string{"a", "b", "bias"}, defaultOutput},
		{"Difference", nil, []string{"a", "b"}, defaultOutput},
		{"Direct", nil, []string{"input"}, defaultOutput},
		{"Distort", nil, []string{"input", "gain", "offsetA", "offsetB"}, defaultOutput},
		{"Divide", nil, []string{"a", "b"}, defaultOutput},
		{"FBComb", nil, []string{"input", "duration", "gain"}, defaultOutput},
		{"FFComb", nil, []string{"input", "duration", "gain"}, defaultOutput},
		{"FileSource", Config{"path": "test/dummy_source.txt"}, nil, defaultOutput},
		{"FilteredFBComb", nil, []string{"input", "gain", "duration", "cutoff", "resonance"}, defaultOutput},
		{"FilteredReverb", nil, []string{"input", "gain", "feedback"}, defaultOutput},
		{"Fold", nil, []string{"input", "level"}, defaultOutput},
		{"GateSequence", Config{"steps": 2}, []string{
			"clock",
			"reset",
			"0.mode",
			"1.mode",
		}, []string{"on", "off"}},
		{"Glide", nil, []string{"input", "rise", "fall"}, defaultOutput},
		{"HPFilter", nil, []string{"input", "cutoff", "resonance"}, defaultOutput},
		{"Interpolate", nil, []string{"input", "min", "max"}, defaultOutput},
		{"Invert", nil, []string{"input"}, defaultOutput},
		{"LPFilter", nil, []string{"input", "cutoff", "resonance"}, defaultOutput},
		{"Mix", nil, []string{
			"0.input",
			"0.level",
			"1.input",
			"1.level",
			"2.input",
			"2.level",
			"3.input",
			"3.level",
			"master",
		}, defaultOutput},
		{"Mod", nil, []string{"a", "b"}, defaultOutput},
		{"Multiple", nil, []string{"input"}, []string{"0", "1", "2", "3"}},
		{"Multiply", nil, []string{"a", "b"}, defaultOutput},
		{"Noise", nil, []string{"input", "max"}, defaultOutput},
		{"OR", nil, []string{"a", "b"}, defaultOutput},
		{"Osc", nil, []string{
			"pitch",
			"pitchMod",
			"pitchModAmount",
			"amp",
			"detune",
			"offset",
			"sync",
		}, []string{
			"sine", "saw", "pulse", "triangle", "sub",
		}},
		{"Oscillator", nil, []string{
			"pitch",
			"pitchMod",
			"pitchModAmount",
			"amp",
			"detune",
			"offset",
			"sync",
		}, []string{
			"sine", "saw", "pulse", "triangle", "sub",
		}},
		{"Quantize", Config{"size": 2}, []string{"0.pitch", "1.pitch"}, defaultOutput},
		{"RandomSeries", nil, []string{
			"clock",
			"max",
			"min",
			"size",
			"trigger",
		}, []string{"gate", "values"}},
		{"Reverb", nil, []string{"input", "gain", "feedback"}, defaultOutput},
		{"SampleHold", nil, []string{"input", "trigger"}, defaultOutput},
		{"Sequence", Config{"stages": 2}, []string{
			"clock",
			"glide",
			"mode",
			"reset",
			"transpose",
			"0.glide",
			"0.mode",
			"0.pitch",
			"0.pulses",
			"1.glide",
			"1.mode",
			"1.pitch",
			"1.pulses",
		}, []string{"gate", "pitch", "sync"}},
		{"Sum", nil, []string{"a", "b"}, defaultOutput},
		{"Switch", nil, []string{"clock", "reset"}, defaultOutput},
		{"Tape", nil, []string{
			"input",
			"play",
			"record",
			"reset",
			"bias",
			"organize",
			"splice",
			"unsplice",
		}, []string{"output", "endOfSplice"}},
		{"TempoDetect", nil, []string{"tap"}, defaultOutput},
		{"Wrap", nil, []string{"input", "level"}, defaultOutput},
		{"XOR", nil, []string{"a", "b"}, defaultOutput},
	}

	moduleNames := map[string]struct{}{}
	for _, n := range RegisteredTypes() {
		moduleNames[n] = struct{}{}
	}

	for _, m := range modules {
		t.Run(m.Name, func(t *testing.T) {
			init, err := Lookup(m.Name)
			assert.Equal(t, err, nil)

			p, err := init(m.Config)
			assert.Equal(t, err, nil)

			mock := &IO{}
			err = mock.Expose(
				[]*In{},
				[]*Out{{Name: "output", Provider: Provide(mockOutput{})}},
			)
			assert.Equal(t, err, nil)

			for _, name := range m.Inputs {
				out, err := mock.Output("output")
				assert.Equal(t, err, nil)

				err = p.Patch(name, out)
				assert.Equal(t, err, nil)
				assert.Equal(t, mock.OutputsActive(), 1)

				err = p.Reset()
				assert.Equal(t, err, nil)
				assert.Equal(t, mock.OutputsActive(), 0)
			}

			frame := make(Frame, FrameSize)
			for _, name := range m.Outputs {
				out, err := p.Output(name)
				assert.Equal(t, err, nil)
				out.Read(frame)
			}
		})

		delete(moduleNames, m.Name)
	}

	if len(moduleNames) != 0 {
		var keys []string
		for k := range moduleNames {
			keys = append(keys, k)
		}
		t.Errorf("no patch tests for %v", strings.Join(keys, ", "))
	}
}
