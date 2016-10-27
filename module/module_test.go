package module

import "testing"

var defaultOutput = []string{"output"}

func TestForExplosions(t *testing.T) {
	modules := []struct {
		Name        string
		OutputNames []string
	}{
		{"ADSR", defaultOutput},
		{"Allpass", defaultOutput},
		{"BinaryDifference", defaultOutput},
		{"BinaryDivide", defaultOutput},
		{"BinaryMultiply", defaultOutput},
		{"BinarySum", defaultOutput},
		{"Clip", defaultOutput},
		{"Compress", defaultOutput},
		{"Direct", defaultOutput},
		{"FBComb", defaultOutput},
		{"FFComb", defaultOutput},
		{"FilteredDelay", defaultOutput},
		{"Fold", defaultOutput},
		{"LPFilter", defaultOutput},
		{"HPFilter", defaultOutput},
		{"Glide", defaultOutput},
		{"Mix", defaultOutput},
		{"Noise", defaultOutput},
		{"Osc", []string{"sine", "saw", "triangle", "pulse"}},
		{"Reverb", defaultOutput},
		{"Multiple", []string{"0", "1", "2", "3"}},
		{"SampleHold", defaultOutput},
		{"Sequence", []string{"gate", "pitch", "sync"}},
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
