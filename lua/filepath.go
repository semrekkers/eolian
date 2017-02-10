package lua

import (
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
)

func preloadFilepath(state *lua.LState) int {
	mod := state.NewTable()
	state.SetFuncs(mod, map[string]lua.LGFunction{
		"dir": dir,
	})
	state.Push(mod)
	return 1
}

func dir(state *lua.LState) int {
	state.Push(lua.LString(filepath.Dir(state.CheckString(1))))
	return 1
}
