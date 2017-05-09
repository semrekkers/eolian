package lua

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"

	"buddin.us/eolian/dsp"
)

var valueFuncs = map[string]lua.LGFunction{
	"bpm":   bpm,
	"hz":    hz,
	"ms":    ms,
	"pitch": pitch,
}

func hz(state *lua.LState) int {
	value := state.ToNumber(1)
	hz := dsp.Frequency(float64(value))

	methods := state.NewTable()
	state.SetFuncs(methods, valuerMethods)
	mt := state.NewTable()
	mt.RawSetString("__index", methods)

	state.Push(&lua.LUserData{Value: hz, Metatable: mt})
	return 1
}

func bpm(state *lua.LState) int {
	value := state.ToNumber(1)
	bpm := dsp.BPM(float64(value))

	methods := state.NewTable()
	state.SetFuncs(methods, valuerMethods)
	mt := state.NewTable()
	mt.RawSetString("__index", methods)

	state.Push(&lua.LUserData{Value: bpm, Metatable: mt})
	return 1
}

func pitch(state *lua.LState) int {
	value := state.ToString(1)
	pitch, err := dsp.ParsePitch(value)
	if err != nil {
		state.RaiseError("%s", err.Error())
	}

	methods := state.NewTable()
	state.SetFuncs(methods, valuerMethods)
	mt := state.NewTable()
	mt.RawSetString("__index", methods)

	state.Push(&lua.LUserData{Value: pitch, Metatable: mt})
	return 1
}

func ms(state *lua.LState) int {
	value := state.ToNumber(1)
	ms := dsp.Duration(float64(value))

	methods := state.NewTable()
	state.SetFuncs(methods, valuerMethods)
	mt := state.NewTable()
	mt.RawSetString("__index", methods)

	state.Push(&lua.LUserData{Value: ms, Metatable: mt})
	return 1
}

var valuerMethods = map[string]lua.LGFunction{
	"value": func(state *lua.LState) int {
		pitch := state.CheckUserData(1).Value.(dsp.Valuer)
		state.Push(lua.LNumber(pitch.Value()))
		return 1
	},
	"string": func(state *lua.LState) int {
		pitch := state.CheckUserData(1).Value.(fmt.Stringer)
		state.Push(lua.LString(pitch.String()))
		return 1
	},
}
