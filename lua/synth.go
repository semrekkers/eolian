package lua

import (
	"fmt"
	"io"
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
			"close":     lock(moduleClose, mtx, p),
			"reset":     lock(moduleReset, mtx, p),
			"resetOnly": lock(moduleResetOnly, mtx, p),
			"set":       lock(moduleSet, mtx, p),
			"id":        lock(moduleID, mtx, p),
			"inputs":    lock(moduleInputs, mtx, p),
			"outputs":   lock(moduleOutputs, mtx, p),

			// Methods that don't need to lock the graph
			"scope": moduleScopedOutput(p),
			"ns":    moduleScopedOutput(p),
			"out":   moduleOutput(p),
			"outFn": moduleOutputFunc(p),
		}
	}(p)

	table := state.NewTable()
	state.RawSet(table, lua.LString("__namespace"), state.NewTable())
	state.RawSet(table, lua.LString("__type"), lua.LString("module"))
	state.SetFuncs(table, funcs)

	return table
}

func moduleInputs(state *lua.LState, p module.Patcher) int {
	l, ok := p.(module.Lister)
	if !ok {
		state.RaiseError("%T is not capable of listing inputs", p)
	}
	t := state.NewTable()
	for k, v := range l.Inputs() {
		t.RawSet(lua.LString(k), lua.LString(v.SourceName()))
	}
	state.Push(t)
	return 1
}

func moduleOutputs(state *lua.LState, p module.Patcher) int {
	l, ok := p.(module.Lister)
	if !ok {
		state.RaiseError("%T is not capable of listing outputs", p)
	}
	t := state.NewTable()
	for k, v := range l.Outputs() {
		t.RawSet(lua.LString(k), lua.LString(v.DestinationName()))
	}
	state.Push(t)
	return 1
}

func moduleSet(state *lua.LState, p module.Patcher) int {
	var (
		self   *lua.LTable
		inputs = map[interface{}]interface{}{}
	)

	top := state.GetTop()
	if top == 2 {
		self = state.CheckTable(1)
		raw := state.CheckTable(2)

		mapped := gluamapper.ToGoValue(raw, mapperOpts)
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
	} else if top == 3 {
		self = state.CheckTable(1)
		name := state.CheckString(2)
		raw := state.CheckAny(3)
		inputs[name] = raw
	} else {
		state.RaiseError("invalid number of arguments to set")
	}

	setInputs(state, p, getNamespace(self), inputs)
	state.Push(self)
	return 1
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

func moduleResetOnly(state *lua.LState, p module.Patcher) int {
	names := state.CheckTable(1)

	inputs := []string{}
	names.ForEach(func(k, v lua.LValue) {
		inputs = append(inputs, v.String())
	})
	if err := p.ResetOnly(inputs); err != nil {
		state.RaiseError("%s", err.Error())
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
	if closer, ok := p.(io.Closer); ok {
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
