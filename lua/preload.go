package lua

import lua "github.com/yuin/gopher-lua"

func preloadLibFile(path string) func(*lua.LState) int {
	return func(state *lua.LState) int {
		content, err := Asset(path)
		if err != nil {
			state.RaiseError(err.Error())
		}
		if err := state.DoString(string(content)); err != nil {
			state.RaiseError(err.Error())
		}
		return 1
	}
}

func loadLibFile(state *lua.LState, path string) error {
	content, err := Asset(path)
	if err != nil {
		return err
	}
	return state.DoString(string(content))
}
