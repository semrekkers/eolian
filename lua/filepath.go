package lua

import (
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
)

func OpenFilePath(state *lua.LState) int {
	fns := map[string]lua.LGFunction{
		"dir": dir,
	}
	module := state.RegisterModule("filepath", fns)
	state.Push(module)
	return 1
}

func dir(state *lua.LState) int {
	state.Push(lua.LString(filepath.Dir(state.CheckString(1))))
	return 1
}
