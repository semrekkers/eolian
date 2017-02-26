package module

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/go-playground/assert.v1"
)

var defaultOutput = []string{"output"}

var allModules = []struct {
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
	}, []string{"output", "endCycle"}},
	{"AND", nil, []string{"a", "b"}, defaultOutput},
	{"Allpass", nil, []string{"input", "duration", "gain"}, defaultOutput},
	{"Clip", nil, []string{"input", "level"}, defaultOutput},
	{"ClockMultiply", nil, []string{"input", "multiplier"}, defaultOutput},
	{"ClockDivide", nil, []string{"input", "divisor"}, defaultOutput},
	{"Compress", nil, []string{"input", "attack", "release"}, defaultOutput},
	{"Crossfade", nil, []string{"a", "b", "bias"}, defaultOutput},
	{"Concurrent", nil, []string{"input"}, defaultOutput},
	{"Control", nil, []string{"control", "mod"}, defaultOutput},
	{"Debug", nil, []string{"input"}, defaultOutput},
	{"Delay", nil, []string{"input", "duration"}, defaultOutput},
	{"Demux", nil, []string{"input", "selection"}, []string{"0", "1"}},
	{"Difference", nil, []string{"a", "b"}, defaultOutput},
	{"Direct", nil, []string{"input"}, defaultOutput},
	{"Distort", nil, []string{"input", "gain", "offsetA", "offsetB"}, defaultOutput},
	{"Divide", nil, []string{"a", "b"}, defaultOutput},
	{"FBComb", nil, []string{"input", "duration", "gain"}, defaultOutput},
	{"FBLoopComb", nil, []string{"input", "duration", "gain"}, defaultOutput},
	{"FFComb", nil, []string{"input", "duration", "gain"}, defaultOutput},
	{"FileSource", Config{"path": "test/dummy_source.txt"}, nil, defaultOutput},
	{"FilteredFBComb", nil, []string{"input", "gain", "duration", "cutoff", "resonance"}, defaultOutput},
	{"FilteredReverb", nil, []string{"input", "gain", "feedback"}, defaultOutput},
	{"Floor", nil, []string{"input"}, defaultOutput},
	{"Fold", nil, []string{"input", "level"}, defaultOutput},
	{"Follow", nil, []string{"input", "attack", "release"}, defaultOutput},
	{"GateMix", nil, []string{
		"0",
		"1",
		"2",
		"3",
	}, defaultOutput},
	{"GateSequence", Config{"steps": 2}, []string{
		"clock",
		"reset",
		"0.mode",
		"1.mode",
	}, []string{"on", "off"}},
	{"Glide", nil, []string{"input", "rise", "fall"}, defaultOutput},
	{"Interpolate", nil, []string{"input", "min", "max"}, defaultOutput},
	{"Invert", nil, []string{"input"}, defaultOutput},
	{"Filter", nil, []string{"input", "cutoff", "resonance"}, []string{"lowpass", "bandpass", "highpass"}},
	{"LPGate", nil, []string{"input", "cutoff", "resonance", "ctrl", "mode"}, defaultOutput},
	{"MathExp", Config{"expression": "x + y * 2"}, []string{"x", "y"}, defaultOutput},
	{"Max", nil, []string{"a", "b"}, defaultOutput},
	{"Min", nil, []string{"a", "b"}, defaultOutput},
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
	{"Mux", nil, []string{"0.input", "1.input", "selection"}, []string{"output"}},
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
	{"RotatingClockDivide", nil, []string{"input", "rotate", "reset"}, []string{
		"1",
		"2",
		"3",
		"4",
		"5",
		"6",
		"7",
		"8",
	}},
	{"SampleHold", nil, []string{"input", "trigger"}, defaultOutput},
	{"StageSequence", Config{"stages": 2}, []string{
		"clock",
		"glide",
		"mode",
		"reset",
		"transpose",
		"0.glide",
		"0.mode",
		"0.pitch",
		"0.pulses",
		"0.velocity",
		"1.glide",
		"1.mode",
		"1.pitch",
		"1.pulses",
		"1.velocity",
	}, []string{"gate", "pitch", "sync", "endstage"}},
	{"StepSequence", Config{"steps": 2, "layers": 1}, []string{
		"clock",
		"a/0/pitch",
		"a/1/pitch",
		"0/enabled",
		"1/enabled",
	}, []string{"a/pitch", "0/gate", "1/gate"}},
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
	}, []string{"output", "endsplice"}},
	{"TempoDetect", nil, []string{"tap"}, defaultOutput},
	{"Toggle", nil, []string{"trigger"}, defaultOutput},
	{"Wrap", nil, []string{"input", "level"}, defaultOutput},
	{"XOR", nil, []string{"a", "b"}, defaultOutput},
}

func TestRegisteredModules(t *testing.T) {
	moduleNames := map[string]struct{}{}
	for _, n := range RegisteredTypes() {
		moduleNames[n] = struct{}{}
	}

	for _, m := range allModules {
		t.Run(m.Name, func(t *testing.T) {
			init, err := Lookup(m.Name)
			assert.Equal(t, err, nil)

			p, err := init(m.Config)
			assert.Equal(t, err, nil)

			mock := &IO{}
			err = mock.Expose(
				"MockModule",
				[]*In{},
				[]*Out{{Name: "output", Provider: Provide(mockOutput{})}},
			)
			assert.Equal(t, err, nil)

			for _, name := range m.Inputs {
				out, err := mock.Output("output")
				assert.Equal(t, err, nil)

				err = p.Patch(name, out)
				assert.Equal(t, err, nil)
				assert.Equal(t, mock.OutputsActive(false), 1)

				err = p.Reset()
				assert.Equal(t, err, nil)
				assert.Equal(t, mock.OutputsActive(false), 0)
			}

			frame := make(Frame, FrameSize)
			for _, name := range m.Outputs {
				out, err := p.Output(name)
				assert.Equal(t, err, nil)
				out.Read(frame)
				assert.Equal(t, out.Close(), nil)
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

func BenchmarkModules(b *testing.B) {
	frame := make(Frame, FrameSize)

	for _, m := range allModules {
		init, err := Lookup(m.Name)
		if err != nil {
			b.Error(err)
		}

		p, err := init(m.Config)
		if err != nil {
			b.Error(err)
		}

		for _, out := range m.Outputs {
			port, err := p.Output(m.Outputs[0])
			if err != nil {
				b.Error(err)
			}

			b.Run(fmt.Sprintf("%s_%s", m.Name, out), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					port.Read(frame)
				}
			})

			if err := port.Close(); err != nil {
				b.Error(err)
			}
		}
	}
}
