package lua

import (
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func preloadString(state *lua.LState) int {
	mod := state.NewTable()
	state.SetFuncs(mod, map[string]lua.LGFunction{
		"split": split,
	})
	state.Push(mod)
	return 1
}

func split(state *lua.LState) int {
	str := state.CheckString(1)
	del := state.CheckString(2)
	t := state.NewTable()
	for _, s := range strings.Split(str, del) {
		t.Append(lua.LString(s))
	}
	state.Push(t)
	return 1
}
