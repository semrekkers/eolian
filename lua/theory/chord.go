package theory

import (
	"buddin.us/musictheory"
	"buddin.us/musictheory/intervals"
	lua "github.com/yuin/gopher-lua"
)

func newChord(state *lua.LState) int {
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
	case "majorTriad", "maj":
		series = intervals.MajorTriad
	case "majorSixth", "maj6":
		series = intervals.MajorSixth
	case "majorSeventh", "maj7":
		series = intervals.MajorSeventh
	case "dominantSeventh":
		series = intervals.DominantSeventh

	case "minorTriad", "min":
		series = intervals.MinorTriad
	case "minorSixth", "min6":
		series = intervals.MinorSixth
	case "minorSeventh", "min7":
		series = intervals.MinorSeventh
	case "halfDiminishedSeventh", "min7b5":
		series = intervals.HalfDiminishedSeventh

	case "diminishedTriad", "dim":
		series = intervals.DiminishedTriad
	case "diminishedSeventh", "dim7":
		series = intervals.DiminishedSeventh
	case "diminishedMajorSeventh", "dimMaj7":
		series = intervals.DiminishedMajorSeventh

	case "augmentedTriad", "aug":
		series = intervals.AugmentedTriad
	case "augmentedSixth", "aug6":
		series = intervals.AugmentedSixth
	case "augmentedSeventh", "aug7":
		series = intervals.AugmentedSeventh
	case "augmentedMajorSeventh", "augMaj7":
		series = intervals.AugmentedMajorSeventh

	default:
		state.RaiseError("unknown scale intervals %s", name)
	}

	state.Push(newChordUserData(state, musictheory.NewChord(root, series)))
	return 1
}

func newChordUserData(state *lua.LState, chord musictheory.Chord) *lua.LUserData {
	methods := state.NewTable()
	state.SetFuncs(methods, map[string]lua.LGFunction{
		"pitches": func(state *lua.LState) int {
			chord := state.CheckUserData(1).Value.(musictheory.Chord)
			t := state.NewTable()
			for _, p := range chord {
				t.Append(newPitchUserData(state, p))
			}
			state.Push(t)
			return 1
		},
		"transpose": func(state *lua.LState) int {
			chord := state.CheckUserData(1).Value.(musictheory.Chord)
			intervalUD := state.CheckUserData(2)
			if interval, ok := intervalUD.Value.(musictheory.Interval); ok {
				state.Push(newChordUserData(state, chord.Transpose(interval).(musictheory.Chord)))
				return 1
			}
			state.RaiseError("argument is not an interval")
			return 1
		},
		"invert": func(state *lua.LState) int {
			chord := state.CheckUserData(1).Value.(musictheory.Chord)
			degree := int(state.CheckNumber(2))
			state.Push(newChordUserData(state, chord.Invert(degree)))
			return 1
		},
		"count": func(state *lua.LState) int {
			chord := state.CheckUserData(1).Value.(musictheory.Chord)
			state.Push(lua.LNumber(len(chord)))
			return 1
		},
	})

	mt := state.NewTable()
	mt.RawSetString("__index", methods)
	return &lua.LUserData{
		Metatable: mt,
		Value:     chord,
	}
}
