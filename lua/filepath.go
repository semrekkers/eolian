package lua

import (
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
)

func openFilePath(state *lua.LState) {
	fns := map[string]lua.LGFunction{
		"dir": dir,
	}
	state.RegisterModule("filepath", fns)
}

func dir(state *lua.LState) int {
	state.Push(lua.LString(filepath.Dir(state.CheckString(1))))
	return 1
}
