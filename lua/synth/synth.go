package synth

import (
	"sync"

	"buddin.us/eolian/module"
	lua "github.com/yuin/gopher-lua"
)

func Preload(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		fns := map[string]lua.LGFunction{}
		for _, name := range module.RegisteredTypes() {
			fns[name] = constructor(name, mtx)
		}
		mod := state.NewTable()
		for k, v := range constants {
			state.SetField(mod, k, v)
		}
		state.SetFuncs(mod, fns)
		state.Push(mod)
		return 1
	}
}
