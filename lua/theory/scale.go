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

	t := state.NewTable()
	scale := musictheory.NewScale(root, series, octaves)
	for i, p := range scale {
		t.RawSetInt(i+1, newPitchUserData(state, p.(musictheory.Pitch)))
	}
	state.Push(t)
	return 1
}
