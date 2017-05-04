package theory

import (
	"buddin.us/eolian/dsp"
	"buddin.us/musictheory"
	lua "github.com/yuin/gopher-lua"
)

func newPitch(state *lua.LState) int {
	p, err := musictheory.ParsePitch(state.CheckString(1))
	if err != nil {
		state.RaiseError("%s", err.Error())
	}
	state.Push(newPitchUserData(state, *p))
	return 1
}

func newPitchUserData(state *lua.LState, p musictheory.Pitch) *lua.LUserData {
	methods := state.NewTable()
	state.SetFuncs(methods, map[string]lua.LGFunction{
		"value": func(state *lua.LState) int {
			pitch := state.CheckUserData(1).Value.(musictheory.Pitch)
			state.Push(&lua.LUserData{
				Value: dsp.Pitch{
					Raw:    pitch.Name(musictheory.AscNames),
					Valuer: dsp.Frequency(pitch.Freq()),
				},
			})
			return 1
		},
		"name": func(state *lua.LState) int {
			pitch := state.CheckUserData(1).Value.(musictheory.Pitch)
			state.Push(lua.LString(pitch.Name(musictheory.AscNames)))
			return 1
		},
		"transpose": func(state *lua.LState) int {
			pitch := state.CheckUserData(1).Value.(musictheory.Pitch)
			userdata := state.CheckUserData(2)
			if interval, ok := userdata.Value.(musictheory.Interval); ok {
				state.Push(newPitchUserData(state, pitch.Transpose(interval).(musictheory.Pitch)))
				return 1
			}
			state.RaiseError("argument is not an interval")
			return 0
		},
	})

	mt := state.NewTable()
	mt.RawSetString("__index", methods)
	return &lua.LUserData{
		Metatable: mt,
		Value:     p,
	}
}
