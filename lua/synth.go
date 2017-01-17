package lua

import (
	"fmt"
	"strings"

	"github.com/brettbuddin/eolian/module"
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

func preloadSynth(p module.Patcher) lua.LGFunction {
	return func(state *lua.LState) int {
		fns := map[string]lua.LGFunction{}
		for name, t := range module.Registry {
			fns[name] = buildConstructor(t)
		}
		state.Push(state.SetFuncs(state.NewTable(), fns))
		return 1
	}
}

func buildConstructor(init module.InitFunc) func(state *lua.LState) int {
	return func(state *lua.LState) int {
		config := module.Config{}

		if state.GetTop() > 0 {
			table := state.CheckTable(1)
			fields := gluamapper.ToGoValue(table, mapperOpts).(map[interface{}]interface{})
			for k, v := range fields {
				switch asserted := v.(type) {
				case *lua.LUserData:
					switch ud := asserted.Value.(type) {
					case module.Valuer:
						config[k.(string)] = ud.Value()
					default:
						state.RaiseError("unconvertible userdata assigned: %T", ud)
					}
				default:
					config[k.(string)] = v
				}
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
	raw := table.RawGet(lua.LString("__namespace")).(*lua.LTable)
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
			"info": func(state *lua.LState) int {
				str := "(no info)"
				if v, ok := p.(module.Inspecter); ok {
					str = v.Inspect()
				}
				state.Push(lua.LString(str))
				return 1
			},
			"inspect": func(state *lua.LState) int {
				if v, ok := p.(module.Inspecter); ok {
					fmt.Println(v.Inspect())
				}
				return 0
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
			"outputFn": func(state *lua.LState) int {
				self := state.CheckTable(1)
				name := "output"
				if state.GetTop() > 1 {
					name = state.ToString(2)
				}

				namespace := getNamespace(self)
				name = strings.Join(append(namespace, name), ".")

				fn := state.NewFunction(lua.LGFunction(func(state *lua.LState) int {
					state.Push(&lua.LUserData{Value: module.Port{p, name}})
					return 1
				}))
				state.Push(fn)
				return 1
			},
			"set": func(state *lua.LState) int {
				var (
					self   *lua.LTable
					prefix []string
					raw    *lua.LTable
				)

				top := state.GetTop()
				if top == 2 {
					self = state.CheckTable(1)
					raw = state.CheckTable(2)
				} else if top == 3 {
					self = state.CheckTable(1)
					prefix = strings.Split(state.CheckAny(2).String(), ".")
					raw = state.CheckTable(3)
				}

				namespace := append(getNamespace(self), prefix...)

				mapped := gluamapper.ToGoValue(raw, mapperOpts)
				inputs, ok := mapped.(map[interface{}]interface{})
				if !ok {
					state.RaiseError("expected table, but got %T instead", mapped)
				}
				setInputs(state, p, namespace, inputs)
				return 0
			},
			"scope": scopedOutput(p),
			"ns":    scopedOutput(p),
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
	state.RawSet(table, lua.LString("__namespace"), state.NewTable())
	state.RawSet(table, lua.LString("__type"), lua.LString("module"))
	state.SetFuncs(table, funcs)

	return table
}

func setInputs(state *lua.LState, p module.Patcher, namespace []string, inputs map[interface{}]interface{}) {
	for key, raw := range inputs {
		full := append(namespace, key.(string))

		if inputs, ok := raw.(map[interface{}]interface{}); ok {
			setInputs(state, p, full, inputs)
			continue
		}

		name := strings.Join(full, ".")

		switch v := raw.(type) {
		case *lua.LUserData:
			switch mv := v.Value.(type) {
			case module.Patcher:
				if err := p.Patch(name, mv); err != nil {
					state.RaiseError("%s", err.Error())
				}
			case module.Valuer:
				if err := p.Patch(name, mv); err != nil {
					state.RaiseError("%s", err.Error())
				}
			default:
				state.RaiseError("not a patcher (%T)", mv)
			}
		case lua.LNumber:
			if err := p.Patch(name, float64(v)); err != nil {
				state.RaiseError("%s", err.Error())
			}
		default:
			if err := p.Patch(name, raw); err != nil {
				state.RaiseError("%s", err.Error())
			}
		}
	}
}

func scopedOutput(p module.Patcher) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)

		newNamespace := state.NewTable()
		namespace := self.RawGet(lua.LString("__namespace")).(*lua.LTable)
		namespace.ForEach(func(_, v lua.LValue) { newNamespace.Append(v) })

		name := state.ToString(2)
		newNamespace.Append(lua.LString(name))

		proxy := state.NewTable()
		proxy.RawSet(lua.LString("__namespace"), newNamespace)
		mt := state.NewTable()
		mt.RawSet(lua.LString("__index"), self)
		state.SetMetatable(proxy, mt)
		state.Push(proxy)
		return 1
	}
}
