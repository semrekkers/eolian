package lua

import (
	"runtime"

	lua "github.com/yuin/gopher-lua"
)

func preloadRuntime(state *lua.LState) int {
	mod := state.NewTable()
	state.SetFuncs(mod, map[string]lua.LGFunction{
		"numgoroutine": numGoroutine,
	})
	state.Push(mod)
	return 1
}

func numGoroutine(state *lua.LState) int {
	state.Push(lua.LNumber(runtime.NumGoroutine()))
	return 1
}
