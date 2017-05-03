package theory

import (
	"buddin.us/musictheory"
	lua "github.com/yuin/gopher-lua"
)

func newPerfectInterval(state *lua.LState) int {
	state.Push(&lua.LUserData{
		Value: musictheory.Perfect(int(state.CheckNumber(1))),
	})
	return 1
}

func newAugmentedInterval(state *lua.LState) int {
	state.Push(&lua.LUserData{
		Value: musictheory.Augmented(int(state.CheckNumber(1))),
	})
	return 1
}

func newDoublyAugmentedInterval(state *lua.LState) int {
	state.Push(&lua.LUserData{
		Value: musictheory.DoublyAugmented(int(state.CheckNumber(1))),
	})
	return 1
}

func newMajorInterval(state *lua.LState) int {
	state.Push(&lua.LUserData{
		Value: musictheory.Major(int(state.CheckNumber(1))),
	})
	return 1
}

func newMinorInterval(state *lua.LState) int {
	state.Push(&lua.LUserData{
		Value: musictheory.Minor(int(state.CheckNumber(1))),
	})
	return 1
}

func newDiminishedInterval(state *lua.LState) int {
	state.Push(&lua.LUserData{
		Value: musictheory.Diminished(int(state.CheckNumber(1))),
	})
	return 1
}

func newDoublyDiminishedInterval(state *lua.LState) int {
	state.Push(&lua.LUserData{
		Value: musictheory.DoublyDiminished(int(state.CheckNumber(1))),
	})
	return 1
}

func newOctaveInterval(state *lua.LState) int {
	state.Push(&lua.LUserData{
		Value: musictheory.Octave(int(state.CheckNumber(1))),
	})
	return 1
}
