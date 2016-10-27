package lua

import (
	"strings"

	"github.com/brettbuddin/eolian/module"
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

func OpenSynth(state *lua.LState, p module.Patcher) int {
	fns := map[string]lua.LGFunction{}
	for name, t := range module.Registry {
		fns[name] = buildConstructor(t)
	}
	module := state.RegisterModule("synth", fns)
	state.SetField(module, "Engine", decoratePatcher(state, p))
	state.Push(module)
	return 1
}

func buildConstructor(init module.InitFunc) func(state *lua.LState) int {
	return func(state *lua.LState) int {
		config := module.Config{}

		if state.GetTop() > 0 {
			table := state.CheckTable(1)
			fields := gluamapper.ToGoValue(table, mapperOpts).(map[interface{}]interface{})
			for k, v := range fields {
				config[k.(string)] = v
			}
		}

		p, err := init(config)
		if err != nil {
			state.RaiseError("%s", err.Error())
		}

		table := decoratePatcher(state, p)
		state.Push(table)
		return 1
	}
}

func getNamespace(table *lua.LTable) []string {
	raw := table.RawGet(lua.LString("_namespace")).(*lua.LTable)
	namespace := gluamapper.ToGoValue(raw, mapperOpts)

	segs := []string{}
	if ns, ok := namespace.([]interface{}); ok {
		for _, v := range ns {
			segs = append(segs, v.(string))
		}
	}
	return segs
}

func decoratePatcher(state *lua.LState, p module.Patcher) *lua.LTable {
	funcs := func(p module.Patcher) map[string]lua.LGFunction {
		return map[string]lua.LGFunction{
			"inspect": func(state *lua.LState) int {
				str := "(no info)"
				if v, ok := p.(module.Inspecter); ok {
					str = v.Inspect()
				}
				state.Push(lua.LString(str))
				return 1
			},
			"output": func(state *lua.LState) int {
				self := state.CheckTable(1)

				name := "output"
				if state.GetTop() > 1 {
					name = state.ToString(2)
				}

				namespace := getNamespace(self)
				name = strings.Join(append(namespace, name), ".")

				state.Push(&lua.LUserData{Value: module.Port{p, name}})
				return 1
			},
			"set": func(state *lua.LState) int {
				self := state.CheckTable(1)
				raw := state.CheckTable(2)
				namespace := getNamespace(self)
				inputs := gluamapper.ToGoValue(raw, mapperOpts).(map[interface{}]interface{})

				for key, raw := range inputs {
					name := strings.Join(append(namespace, key.(string)), ".")

					if ud, ok := raw.(*lua.LUserData); ok {
						if out, ok := ud.Value.(module.Patcher); ok {
							if err := p.Patch(name, out); err != nil {
								state.RaiseError("%s", err.Error())
							}
						} else if v, ok := ud.Value.(module.Valuer); ok {
							if err := p.Patch(name, v); err != nil {
								state.RaiseError("%s", err.Error())
							}
						} else {
							state.RaiseError("not a patcher (%T)", ud.Value)
						}
					} else if n, ok := raw.(lua.LNumber); ok {
						if err := p.Patch(name, float64(n)); err != nil {
							state.RaiseError("%s", err.Error())
						}
					} else {
						if err := p.Patch(name, raw); err != nil {
							state.RaiseError("%s", err.Error())
						}
					}
				}
				return 0
			},
			"scope": func(state *lua.LState) int {
				self := state.CheckTable(1)

				newNamespace := state.NewTable()
				namespace := self.RawGet(lua.LString("_namespace")).(*lua.LTable)
				namespace.ForEach(func(_, v lua.LValue) { newNamespace.Append(v) })

				name := state.ToString(2)
				newNamespace.Append(lua.LString(name))

				proxy := state.NewTable()
				proxy.RawSet(lua.LString("_namespace"), newNamespace)
				mt := state.NewTable()
				mt.RawSet(lua.LString("__index"), self)
				state.SetMetatable(proxy, mt)
				state.Push(proxy)
				return 1
			},
			"reset": func(state *lua.LState) int {
				if err := p.Reset(); err != nil {
					state.RaiseError("%s", err.Error())
				}
				return 0
			},
			"close": func(state *lua.LState) int {
				if closer, ok := p.(module.Closer); ok {
					if err := closer.Close(); err != nil {
						state.RaiseError("%s", err.Error())
					}
				}
				return 0
			},
		}
	}(p)

	table := state.NewTable()
	state.RawSet(table, lua.LString("_namespace"), state.NewTable())
	state.RawSet(table, lua.LString("_type"), lua.LString("module"))
	state.SetFuncs(table, funcs)

	return table
}
