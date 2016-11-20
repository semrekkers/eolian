package module

import "testing"

var defaultOutput = []string{"output"}

func TestDefaultsForExplosions(t *testing.T) {
	modules := []struct {
		Name        string
		Config      Config
		OutputNames []string
	}{
		{"ADSR", nil, defaultOutput},
		{"AND", nil, defaultOutput},
		{"Allpass", nil, defaultOutput},
		{"Clip", nil, defaultOutput},
		{"Compress", nil, defaultOutput},
		{"Crossfade", nil, defaultOutput},
		{"Difference", nil, defaultOutput},
		{"Direct", nil, defaultOutput},
		{"Distort", nil, defaultOutput},
		{"Divide", nil, defaultOutput},
		{"FBComb", nil, defaultOutput},
		{"FFComb", nil, defaultOutput},
		{"FileSource", Config{"path": "test/dummy_source.txt"}, defaultOutput},
		{"FilteredDelay", nil, defaultOutput},
		{"Fold", nil, defaultOutput},
		{"Glide", nil, defaultOutput},
		{"HPFilter", nil, defaultOutput},
		{"Interpolate", nil, defaultOutput},
		{"Invert", nil, defaultOutput},
		{"LPFilter", nil, defaultOutput},
		{"Tape", nil, defaultOutput},
		{"Mix", nil, defaultOutput},
		{"Mod", nil, defaultOutput},
		{"Multiple", nil, []string{"0", "1", "2", "3"}},
		{"Multiply", nil, defaultOutput},
		{"Noise", nil, defaultOutput},
		{"OR", nil, defaultOutput},
		{"Osc", nil, []string{"sine", "saw", "triangle", "pulse"}},
		{"RandomSeries", nil, []string{"values", "gate"}},
		{"Reverb", nil, defaultOutput},
		{"SampleHold", nil, defaultOutput},
		{"Sequence", nil, []string{"gate", "pitch", "sync"}},
		{"Sum", nil, defaultOutput},
		{"Switch", nil, defaultOutput},
		{"Wrap", nil, defaultOutput},
		{"XOR", nil, defaultOutput},
	}

	for _, m := range modules {
		init, err := Lookup(m.Name)
		if err != nil {
			t.Error(err)
		}

		p, err := init(m.Config)
		if err != nil {
			t.Error(err)
		}

		for _, name := range m.OutputNames {
			out, err := p.Output(name)
			if err != nil {
				t.Error(err)
			}
			out.Read(make(Frame, FrameSize))
		}
	}
}
