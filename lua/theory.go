package lua

import (
	"buddin.us/eolian/dsp"
	"buddin.us/musictheory"
	"buddin.us/musictheory/intervals"
	lua "github.com/yuin/gopher-lua"
)

func preloadTheory(state *lua.LState) int {
	module := state.RegisterModule("theory", theoryFuncs)
	state.Push(module)
	return 1
}

var theoryFuncs = map[string]lua.LGFunction{
	"pitch":            newPitch,
	"perfect":          newPerfectInterval,
	"augmented":        newAugmentedInterval,
	"doublyAugmented":  newDoublyAugmentedInterval,
	"major":            newMajorInterval,
	"minor":            newMinorInterval,
	"diminished":       newDiminishedInterval,
	"doublyDiminished": newDoublyDiminishedInterval,
	"octave":           newOctaveInterval,
	"scale":            newScale,
}

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

	var (
		scale = musictheory.NewScale(root, series, octaves)
		t     = state.NewTable()
	)
	for i, v := range scale {
		p := v.(musictheory.Pitch)
		ud := state.NewUserData()
		ud.Value = dsp.Pitch{Raw: p.Name(musictheory.AscNames), Valuer: dsp.Frequency(p.Freq())}
		t.RawSetInt(i+1, ud)
	}
	state.Push(t)
	return 1
}

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

func addPitchMethods(state *lua.LState, table *lua.LTable, p musictheory.Pitch) {
	funcs := func(p musictheory.Pitch) map[string]lua.LGFunction {
		return map[string]lua.LGFunction{
			"value": func(state *lua.LState) int {
				v := dsp.Pitch{Raw: p.Name(musictheory.AscNames), Valuer: dsp.Frequency(p.Freq())}
				state.Push(&lua.LUserData{Value: v})
				return 1
			},
			"name": func(state *lua.LState) int {
				state.Push(lua.LString(p.Name(musictheory.AscNames)))
				return 1
			},
			"transpose": func(state *lua.LState) int {
				userdata := state.CheckUserData(2)
				if interval, ok := userdata.Value.(musictheory.Interval); ok {
					table := state.NewTable()
					p := p.Transpose(interval).(musictheory.Pitch)
					addPitchMethods(state, table, p)
					state.Push(table)
					return 1
				}
				state.RaiseError("userdata not an interval")
				return 0
			},
		}
	}(p)

	state.SetFuncs(table, funcs)
}

func newPitch(state *lua.LState) int {
	p, err := musictheory.ParsePitch(state.CheckString(1))
	if err != nil {
		state.RaiseError("%s", err.Error())
	}

	table := state.NewTable()
	addPitchMethods(state, table, *p)
	state.Push(table)
	return 1
}
