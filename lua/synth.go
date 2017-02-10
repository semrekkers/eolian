package lua

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"

	"github.com/brettbuddin/eolian/module"
)

var mapperOpts = gluamapper.Option{
	NameFunc: func(v string) string {
		return v
	},
}

var synthConsts = map[string]lua.LValue{
	// Sequencer gate modes
	"MODE_REST":   lua.LNumber(0),
	"MODE_SINGLE": lua.LNumber(1),
	"MODE_REPEAT": lua.LNumber(2),
	"MODE_HOLD":   lua.LNumber(3),

	// Sequencer patterns
	"MODE_SEQUENTIAL": lua.LNumber(0),
	"MODE_PINGPONG":   lua.LNumber(1),
	"MODE_RANDOM":     lua.LNumber(2),

	// Gate sequencer modes
	"MODE_OFF": lua.LNumber(0),
	"MODE_ON":  lua.LNumber(1),
}

func preloadSynth(mtx *sync.Mutex) lua.LGFunction {
	return func(state *lua.LState) int {
		fns := map[string]lua.LGFunction{}
		for name, t := range module.Registry {
			fns[name] = buildConstructor(name, t, mtx)
		}
		mod := state.NewTable()
		for k, v := range synthConsts {
			state.SetField(mod, k, v)
		}
		state.SetFuncs(mod, fns)
		state.Push(mod)
		return 1
	}
}

func buildConstructor(name string, init module.InitFunc, mtx *sync.Mutex) func(state *lua.LState) int {
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

		table := decoratePatcher(state, p, mtx)
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

type lockingModuleMethod func(state *lua.LState, p module.Patcher) int

func lock(m lockingModuleMethod, mtx *sync.Mutex, p module.Patcher) lua.LGFunction {
	return func(state *lua.LState) int {
		mtx.Lock()
		defer mtx.Unlock()
		return m(state, p)
	}
}

func decoratePatcher(state *lua.LState, p module.Patcher, mtx *sync.Mutex) *lua.LTable {
	funcs := func(p module.Patcher) map[string]lua.LGFunction {
		return map[string]lua.LGFunction{
			// Methods lock and interact with the graph
			"close":       lock(moduleClose, mtx, p),
			"info":        lock(moduleInfo, mtx, p),
			"inspect":     lock(moduleInspect, mtx, p),
			"reset":       lock(moduleReset, mtx, p),
			"set":         lock(moduleSet, mtx, p),
			"id":          lock(moduleID, mtx, p),
			"inputNames":  lock(moduleInputNames, mtx, p),
			"outputNames": lock(moduleOutputNames, mtx, p),

			// Methods that don't need to lock the graph
			"scope":    moduleScopedOutput(p),
			"ns":       moduleScopedOutput(p),
			"output":   moduleOutput(p),
			"outputFn": moduleOutputFunc(p),
		}
	}(p)

	table := state.NewTable()
	state.RawSet(table, lua.LString("__namespace"), state.NewTable())
	state.RawSet(table, lua.LString("__type"), lua.LString("module"))
	state.SetFuncs(table, funcs)

	return table
}

func moduleInputNames(state *lua.LState, p module.Patcher) int {
	l, ok := p.(module.Lister)
	if !ok {
		state.RaiseError("%T is not capable of listing inputs", p)
	}
	t := state.NewTable()
	for in := range l.Inputs() {
		t.Append(lua.LString(in))
	}
	state.Push(t)
	return 1
}

func moduleOutputNames(state *lua.LState, p module.Patcher) int {
	l, ok := p.(module.Lister)
	if !ok {
		state.RaiseError("%T is not capable of listing outputs", p)
	}
	t := state.NewTable()
	for in := range l.Outputs() {
		t.Append(lua.LString(in))
	}
	state.Push(t)
	return 1
}

func moduleSet(state *lua.LState, p module.Patcher) int {
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
		prefix = strings.Split(state.CheckAny(2).String(), "/")
		raw = state.CheckTable(3)
	}

	namespace := append(getNamespace(self), prefix...)

	mapped := gluamapper.ToGoValue(raw, mapperOpts)

	inputs := map[interface{}]interface{}{}
	switch v := mapped.(type) {
	case map[interface{}]interface{}:
		inputs = v
	case []interface{}:
		for i, rv := range v {
			inputs[fmt.Sprintf("%d", i)] = rv
		}
	default:
		state.RaiseError("expected table, but got %T instead", mapped)
	}
	setInputs(state, p, namespace, inputs)
	return 0
}

func setInputs(state *lua.LState, p module.Patcher, namespace []string, inputs map[interface{}]interface{}) {
	for key, raw := range inputs {
		full := append(namespace, key.(string))

		if inputs, ok := raw.(map[interface{}]interface{}); ok {
			setInputs(state, p, full, inputs)
			continue
		}

		name := strings.Join(full, "/")

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

func moduleID(state *lua.LState, p module.Patcher) int {
	state.Push(lua.LString(p.ID()))
	return 1
}

func moduleInfo(state *lua.LState, p module.Patcher) int {
	str := "(no info)"
	if v, ok := p.(module.Inspecter); ok {
		str = v.Inspect()
	}
	state.Push(lua.LString(str))
	return 1
}

func moduleInspect(state *lua.LState, p module.Patcher) int {
	if v, ok := p.(module.Inspecter); ok {
		fmt.Println(v.Inspect())
	}
	return 0
}

func moduleReset(state *lua.LState, p module.Patcher) int {
	if err := p.Reset(); err != nil {
		state.RaiseError("%s", err.Error())
	}
	return 0
}

func moduleClose(state *lua.LState, p module.Patcher) int {
	if closer, ok := p.(module.Closer); ok {
		if err := closer.Close(); err != nil {
			state.RaiseError("%s", err.Error())
		}
	}
	return 0
}

func moduleScopedOutput(p module.Patcher) lua.LGFunction {
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

func moduleOutput(p module.Patcher) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		var name string
		if state.GetTop() > 1 {
			name = state.ToString(2)
		}
		if len(name) == 0 {
			name = "output"
		}

		namespace := getNamespace(self)
		name = strings.Join(append(namespace, name), "/")

		state.Push(&lua.LUserData{Value: module.Port{p, name}})
		return 1
	}
}

func moduleOutputFunc(p module.Patcher) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		name := "output"
		if state.GetTop() > 1 {
			name = state.ToString(2)
		}

		namespace := getNamespace(self)
		name = strings.Join(append(namespace, name), "/")

		fn := state.NewFunction(lua.LGFunction(func(state *lua.LState) int {
			state.Push(&lua.LUserData{Value: module.Port{p, name}})
			return 1
		}))
		state.Push(fn)
		return 1
	}
}
