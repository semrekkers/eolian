package lua

import (
	"time"

	lua "github.com/yuin/gopher-lua"
)

func preloadTime(state *lua.LState) int {
	mod := state.NewTable()
	state.SetFuncs(mod, map[string]lua.LGFunction{
		"sleep": timeSleep,
	})
	state.Push(mod)
	return 1
}

func timeSleep(state *lua.LState) int {
	ms := state.CheckNumber(1)
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return 0
}
