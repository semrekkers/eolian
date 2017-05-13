package synth

import (
	"buddin.us/eolian/dsp"
	lua "github.com/yuin/gopher-lua"
)

var constants = map[string]lua.LValue{
	"SAMPLE_RATE": lua.LNumber(dsp.SampleRate),

	// Sequencer gate modes
	"MODE_REST":   lua.LNumber(0),
	"MODE_SINGLE": lua.LNumber(1),
	"MODE_REPEAT": lua.LNumber(2),
	"MODE_HOLD":   lua.LNumber(3),

	// Sequencer patterns
	"MODE_SEQUENTIAL": lua.LNumber(0),
	"MODE_PINGPONG":   lua.LNumber(1),
	"MODE_RANDOM":     lua.LNumber(2),

	// Toggle modes
	"MODE_OFF": lua.LNumber(0),
	"MODE_ON":  lua.LNumber(1),

	// LP gate modes
	"MODE_LOWPASS":   lua.LNumber(0),
	"MODE_COMBO":     lua.LNumber(1),
	"MODE_AMPLITUDE": lua.LNumber(2),
}
