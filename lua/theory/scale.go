package theory

import (
	"buddin.us/musictheory"
	"buddin.us/musictheory/intervals"
	lua "github.com/yuin/gopher-lua"
)

func newScale(state *lua.LState) int {
	pitch := state.CheckString(1)
	root, err := musictheory.ParsePitch(pitch)
	if err != nil {
		state.RaiseError(err.Error())
	}
	var (
		name   = state.CheckString(2)
		series []musictheory.Interval
	)
	switch name {
	case "chromatic":
		series = intervals.Chromatic
	case "major":
		series = intervals.Major
	case "minor":
		series = intervals.Minor
	case "majorPentatonic":
		series = intervals.MajorPentatonic
	case "minorPentatonic":
		series = intervals.MinorPentatonic
	case "ionion":
		series = intervals.Ionian
	case "dorian":
		series = intervals.Dorian
	case "phrygian":
		series = intervals.Phrygian
	case "aeolian":
		series = intervals.Aeolian
	case "lydian":
		series = intervals.Lydian
	case "mixolydian":
		series = intervals.Mixolydian
	case "locrian":
		series = intervals.Locrian
	default:
		state.RaiseError("unknown scale intervals %s", name)
	}
	octaves := state.CheckInt(3)

	state.Push(newScaleUserData(state, musictheory.NewScale(root, series, octaves)))
	return 1
}

func newScaleUserData(state *lua.LState, scale musictheory.Scale) *lua.LUserData {
	methods := state.NewTable()
	state.SetFuncs(methods, map[string]lua.LGFunction{
		"pitches": func(state *lua.LState) int {
			scale := state.CheckUserData(1).Value.(musictheory.Scale)
			t := state.NewTable()
			for _, p := range scale {
				t.Append(newPitchUserData(state, p.(musictheory.Pitch)))
			}
			state.Push(t)
			return 1
		},
		"transpose": func(state *lua.LState) int {
			scale := state.CheckUserData(1).Value.(musictheory.Scale)
			intervalUD := state.CheckUserData(2)
			if interval, ok := intervalUD.Value.(musictheory.Interval); ok {
				state.Push(newScaleUserData(state, scale.Transpose(interval).(musictheory.Scale)))
				return 1
			}
			state.RaiseError("argument is not an interval")
			return 1
		},
		"count": func(state *lua.LState) int {
			scale := state.CheckUserData(1).Value.(musictheory.Scale)
			state.Push(lua.LNumber(len(scale)))
			return 1
		},
	})

	mt := state.NewTable()
	mt.RawSetString("__index", methods)
	return &lua.LUserData{
		Metatable: mt,
		Value:     scale,
	}
}
