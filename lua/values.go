package lua

import (
	lua "github.com/yuin/gopher-lua"

	"github.com/brettbuddin/eolian/module"
)

var valueFuncs = map[string]lua.LGFunction{
	"bpm":   bpm,
	"hz":    hz,
	"ms":    ms,
	"pitch": pitch,
}

func hz(state *lua.LState) int {
	value := state.ToNumber(1)
	hz := module.Frequency(float64(value))
	state.Push(lua.LNumber(hz.Value()))
	return 1
}

func bpm(state *lua.LState) int {
	value := state.ToNumber(1)
	bpm := module.BPM(float64(value))
	state.Push(lua.LNumber(bpm.Value()))
	return 1
}

func pitch(state *lua.LState) int {
	value := state.ToString(1)
	pitch, err := module.ParsePitch(value)
	if err != nil {
		state.RaiseError("%s", err.Error())
	}
	state.Push(lua.LNumber(pitch.Value()))
	return 1
}

func ms(state *lua.LState) int {
	value := state.ToNumber(1)
	ms := module.Duration(float64(value))
	state.Push(lua.LNumber(ms.Value()))
	return 1
}
