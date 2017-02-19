package lua

import lua "github.com/yuin/gopher-lua"

func preloadSynthControl(state *lua.LState) int {
	content, err := Asset("lua/lib/control.lua")
	if err != nil {
		state.RaiseError(err.Error())
	}
	if err := state.DoString(string(content)); err != nil {
		state.RaiseError(err.Error())
	}
	return 1
}
