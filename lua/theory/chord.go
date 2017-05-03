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

	t := state.NewTable()
	chord := musictheory.NewChord(root, series)
	for i, p := range chord {
		t.RawSetInt(i+1, newPitchUserData(state, p))
	}
	state.Push(t)
	return 1
}
