package theory

import lua "github.com/yuin/gopher-lua"

func Preload(state *lua.LState) int {
	module := state.RegisterModule("theory", map[string]lua.LGFunction{
		"augmented":        newAugmentedInterval,
		"chord":            newChord,
		"diminished":       newDiminishedInterval,
		"doublyAugmented":  newDoublyAugmentedInterval,
		"doublyDiminished": newDoublyDiminishedInterval,
		"major":            newMajorInterval,
		"minor":            newMinorInterval,
		"octave":           newOctaveInterval,
		"perfect":          newPerfectInterval,
		"pitch":            newPitch,
		"scale":            newScale,
	})
	state.Push(module)
	return 1
}
