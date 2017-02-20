package lua

import lua "github.com/yuin/gopher-lua"

func preloadSynthProxy(state *lua.LState) int {
	mod := state.SetFuncs(state.NewTable(), proxyFuncs)
	state.Push(mod)
	return 1
}

var proxyFuncs = map[string]lua.LGFunction{
	"inputs":  proxyInputs,
	"outputs": proxyOutputs,
}

func proxyInputs(state *lua.LState) int {
	module := state.CheckTable(1)
	fn := state.NewFunction(func(state *lua.LState) int {
		if state.GetTop() == 2 {
			inputs := state.CheckTable(2)
			state.CallByParam(lua.P{
				Fn:      module.RawGet(lua.LString("set")),
				Protect: true,
				NRet:    1,
			}, module, inputs)
		} else if state.GetTop() == 3 {
			name := state.CheckString(2)
			input := state.CheckAny(3)
			state.CallByParam(lua.P{
				Fn:      module.RawGet(lua.LString("set")),
				Protect: true,
				NRet:    1,
			}, module, lua.LString(name), input)
		}
		return 1
	})
	state.Push(fn)
	return 1
}

func proxyOutputs(state *lua.LState) int {
	module := state.CheckTable(1)
	fn := state.NewFunction(func(state *lua.LState) int {
		if state.GetTop() == 1 {
			state.CallByParam(lua.P{
				Fn:      module.RawGet(lua.LString("out")),
				Protect: true,
				NRet:    1,
			}, module)
			return 1
		} else if output := state.CheckAny(2); output != nil {
			state.CallByParam(lua.P{
				Fn:      module.RawGet(lua.LString("out")),
				Protect: true,
				NRet:    1,
			}, module, output)
			return 1
		} else {
			state.CallByParam(lua.P{
				Fn:      module.RawGet(lua.LString("out")),
				Protect: true,
				NRet:    1,
			}, module)
			return 1
		}
	})
	state.Push(fn)
	return 1
}
