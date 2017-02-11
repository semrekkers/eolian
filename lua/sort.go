package lua

import (
	"sort"

	lua "github.com/yuin/gopher-lua"
)

func preloadSort(state *lua.LState) int {
	mod := state.NewTable()
	state.SetFuncs(mod, map[string]lua.LGFunction{
		"strings": sortStrings,
	})
	state.Push(mod)
	return 1
}

func sortStrings(state *lua.LState) int {
	strs := []string{}
	unsorted := state.CheckTable(1)
	unsorted.ForEach(func(_, v lua.LValue) {
		strs = append(strs, v.String())
	})
	sort.Strings(strs)
	t := state.NewTable()
	for _, s := range strs {
		t.Append(lua.LString(s))
	}
	state.Push(t)
	return 1
}
