package module

import "testing"

var defaultOutput = []string{"output"}

func TestForExplosions(t *testing.T) {
	modules := []struct {
		Name        string
		OutputNames []string
	}{
		{"ADSR", defaultOutput},
		{"AND", defaultOutput},
		{"Allpass", defaultOutput},
		{"Clip", defaultOutput},
		{"Compress", defaultOutput},
		{"Crossfade", defaultOutput},
		{"Difference", defaultOutput},
		{"Direct", defaultOutput},
		{"Divide", defaultOutput},
		{"Distort", defaultOutput},
		{"FBComb", defaultOutput},
		{"FFComb", defaultOutput},
		{"FilteredDelay", defaultOutput},
		{"Fold", defaultOutput},
		{"Glide", defaultOutput},
		{"HPFilter", defaultOutput},
		{"Interpolate", defaultOutput},
		{"Invert", defaultOutput},
		{"LPFilter", defaultOutput},
		{"Mix", defaultOutput},
		{"Multiple", []string{"0", "1", "2", "3"}},
		{"Multiply", defaultOutput},
		{"Mod", defaultOutput},
		{"Noise", defaultOutput},
		{"OR", defaultOutput},
		{"Osc", []string{"sine", "saw", "triangle", "pulse"}},
		{"RandomSeries", []string{"values", "gate"}},
		{"Reverb", defaultOutput},
		{"SampleHold", defaultOutput},
		{"Sequence", []string{"gate", "pitch", "sync"}},
		{"Switch", defaultOutput},
		{"Sum", defaultOutput},
		{"Wrap", defaultOutput},
		{"XOR", defaultOutput},
	}

	for _, m := range modules {
		init, err := Lookup(m.Name)
		if err != nil {
			t.Error(err)
		}

		p, err := init(nil)
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
