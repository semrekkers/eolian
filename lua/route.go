package lua

import lua "github.com/yuin/gopher-lua"

func preloadSynthRoute(state *lua.LState) int {
	content, err := Asset("lua/lib/route.lua")
	if err != nil {
		state.RaiseError(err.Error())
	}
	if err := state.DoString(string(content)); err != nil {
		state.RaiseError(err.Error())
	}
	return 1
}
