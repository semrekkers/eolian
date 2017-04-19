package module

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"buddin.us/eolian/dsp"

	"gopkg.in/go-playground/assert.v1"
)

var defaultOutput = []string{"output"}

var allModules = []struct {
	Name            string
	Config          Config
	Inputs, Outputs []string
}{
	{"AHD", nil, []string{"attack", "decay", "hold", "ratio"}, []string{"output", "endcycle"}},
	{"ADSR", nil, []string{"attack", "decay", "sustain", "release", "disableSustain", "ratio"}, []string{"output", "endcycle"}},
	{"AND", nil, []string{"a", "b"}, defaultOutput},
	{"Allpass", nil, []string{"input", "duration", "gain"}, defaultOutput},
	{"Ceil", nil, []string{"input"}, defaultOutput},
	{"ChanceGate", nil, []string{"input", "bias"}, []string{"a", "b"}},
	{"Clip", nil, []string{"input", "level"}, defaultOutput},
	{"Clock", nil, []string{"tempo", "pulseWidth", "shuffle"}, defaultOutput},
	{"ClockMultiply", nil, []string{"input", "multiplier"}, defaultOutput},
	{"ClockDivide", nil, []string{"input", "divisor"}, defaultOutput},
	{"Compress", nil, []string{"input", "attack", "release"}, defaultOutput},
	{"Crossfade", nil, []string{"a", "b", "bias"}, defaultOutput},
	{"Crossfeed", nil, []string{"a", "b", "amount"}, []string{"a", "b"}},
	{"Concurrent", nil, []string{"input"}, defaultOutput},
	{"Control", nil, []string{"input", "mod", "min", "max"}, defaultOutput},
	{"Count", nil, []string{"trigger", "limit", "step", "reset"}, defaultOutput},
	{"Coupler", nil, []string{"input", "toggle", "duration"}, defaultOutput},
	{"Debug", Config{"output": ioutil.Discard}, []string{"input"}, defaultOutput},
	{"Delay", nil, []string{"input", "duration"}, defaultOutput},
	{"Demux", nil, []string{"input", "selection"}, []string{"0", "1"}},
	{"Difference", nil, []string{"a", "b"}, defaultOutput},
	{"Direct", nil, []string{"input"}, defaultOutput},
	{"Distort", nil, []string{"input", "gain", "offsetA", "offsetB"}, defaultOutput},
	{"Divide", nil, []string{"a", "b"}, defaultOutput},
	{"Edges", nil, []string{"input"}, []string{"endrise", "endcycle"}},
	{"FBDelay", nil, []string{"input", "duration", "gain"}, defaultOutput},
	{"FBLoopDelay", nil, []string{"input", "duration", "gain"}, defaultOutput},
	{"FileSource", Config{"path": "testdata/dummy_source.txt"}, nil, defaultOutput},
	{"FilteredFBDelay", nil, []string{"input", "gain", "duration", "cutoff", "resonance"}, defaultOutput},
	{"FilteredReverb", nil, []string{"input", "gain", "feedback", "cutoff", "fbCutoff"}, defaultOutput},
	{"Floor", nil, []string{"input"}, defaultOutput},
	{"Fold", nil, []string{"input", "level", "gain", "stages"}, defaultOutput},
	{"Follow", nil, []string{"input", "attack", "release"}, defaultOutput},
	{"GateMix", nil, []string{"0", "1", "2", "3"}, defaultOutput},
	{"GateSequence", Config{"steps": 2}, []string{"clock", "reset", "0.mode", "1.mode"}, []string{"on", "off"}},
	{"Glide", nil, []string{"input", "rise", "fall"}, defaultOutput},
	{"Interpolate", nil, []string{"input"}, defaultOutput},
	{"Invert", nil, []string{"input"}, defaultOutput},
	{"Filter", nil, []string{"input", "cutoff", "resonance"}, []string{"lowpass", "bandpass", "highpass"}},
	{"LPGate", nil, []string{"input", "cutoff", "resonance", "control", "mode"}, defaultOutput},
	{"MathExp", Config{"expression": "x + y * 2"}, []string{"x", "y"}, defaultOutput},
	{"Max", nil, []string{"a", "b"}, defaultOutput},
	{"Min", nil, []string{"a", "b"}, defaultOutput},
	{"Mix", nil, []string{"0.input", "0.level", "1.input", "1.level", "2.input", "2.level", "3.input", "3.level", "master"}, defaultOutput},
	{"Mux", nil, []string{"0.input", "1.input", "selection"}, []string{"output"}},
	{"Mod", nil, []string{"a", "b"}, defaultOutput},
	{"Multiple", nil, []string{"input"}, []string{"0", "1", "2", "3"}},
	{"Multiply", nil, []string{"a", "b"}, defaultOutput},
	{"Noise", nil, []string{"input", "max"}, defaultOutput},
	{"NoteQuantize", nil, []string{"input", "octave"}, defaultOutput},
	{"OR", nil, []string{"a", "b"}, defaultOutput},
	{"Osc", nil, []string{"pitch", "pitchMod", "pitchModAmount", "amp", "detune", "offset", "sync", "pulseWidth"},
		[]string{"sine", "saw", "pulse", "triangle", "sub"}},
	{"Oscillator", nil, []string{"pitch", "pitchMod", "pitchModAmount", "amp", "detune", "offset", "sync", "pulseWidth"},
		[]string{"sine", "saw", "pulse", "triangle", "sub"}},
	{"Pan", nil, []string{"input", "bias"}, []string{"a", "b"}},
	{"PanMix", nil, []string{"0.input", "0.level", "0.pan", "1.input", "1.level", "1.pan", "2.input", "2.level", "2.pan", "3.input", "3.level", "3.pan", "master"}, []string{"a", "b"}},
	{"FBPingPongDelay", nil, []string{"a", "b", "duration", "gain"}, []string{"a", "b"}},
	{"Quantize", Config{"size": 2}, []string{"input", "0.pitch", "1.pitch", "transpose"}, defaultOutput},
	{"Random", nil, []string{"clock", "max", "min", "probability", "smoothness"}, []string{"stepped", "smooth"}},
	{"RandomSeries", nil, []string{"clock", "max", "min", "size", "trigger"}, []string{"gate", "value"}},
	{"RandomTrigger", nil, []string{"clock", "probability"}, defaultOutput},
	{"Reverb", nil, []string{"input", "gain", "feedback"}, defaultOutput},
	{"RotatingClockDivide", nil, []string{"input", "rotate", "reset"},
		[]string{"1", "2", "3", "4", "5", "6", "7", "8"}},
	{"Round", nil, []string{"input"}, defaultOutput},
	{"SampleHold", nil, []string{"input", "trigger"}, defaultOutput},
	{"Shape", nil, []string{"gate", "trigger", "rise", "fall", "ratio", "cycle"}, []string{"output", "endcycle"}},
	{"SoftClip", nil, []string{"input", "gain"}, defaultOutput},
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
	{"StepSequence", Config{"steps": 2, "layers": 1},
		[]string{"clock", "mode", "a/0/pitch", "a/1/pitch", "0/enabled", "1/enabled"},
		[]string{"a/pitch", "0/gate", "1/gate", "gate"}},
	{"Sum", nil, []string{"a", "b"}, defaultOutput},
	{"Survey", nil, []string{"a", "b", "or1", "or2", "and1", "and2", "slope", "crease", "offset", "fade", "survey"}, []string{"a", "b", "crease", "follow", "or", "and", "slope"}},
	{"Switch", nil, []string{"clock", "reset"}, defaultOutput},
	{"TankReverb", nil, []string{"a", "b"}, []string{"a", "b"}},
	{"Tap", nil, []string{"input", "tap"}, []string{"output", "tap"}},
	{"Tape", nil, []string{"input", "play", "record", "reset", "bias", "organize", "splice", "unsplice", "zoom", "slide"},
		[]string{"output", "endsplice"}},
	{"TempoDetect", nil, []string{"tap"}, defaultOutput},
	{"Toggle", nil, []string{"trigger"}, defaultOutput},
	{"TrackHold", nil, []string{"input", "hang"}, defaultOutput},
	{"VariableRandomSeries", nil, []string{"clock", "max", "min", "size", "random"}, []string{"gate", "value"}},
	{"Wrap", nil, []string{"input", "level"}, defaultOutput},
	{"Wavetable", nil, []string{"pitch", "amp"}, defaultOutput},
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
				[]*Out{{Name: "output", Provider: dsp.Provide(mockOutput{})}},
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

			frame := dsp.NewFrame()
			for _, name := range m.Outputs {
				out, err := p.Output(name)
				assert.Equal(t, err, nil)
				out.Process(frame)
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
	frame := dsp.NewFrame()

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
					port.Process(frame)
				}
			})

			if err := port.Close(); err != nil {
				b.Error(err)
			}
		}
	}
}
