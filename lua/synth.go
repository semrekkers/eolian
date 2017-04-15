package lua

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"

	"buddin.us/eolian/dsp"
	"buddin.us/eolian/module"
)

var mapperOpts = gluamapper.Option{
	NameFunc: func(v string) string {
		return v
	},
}

var synthConsts = map[string]lua.LValue{
	"SAMPLE_RATE": lua.LNumber(dsp.SampleRate),

	// Sequencer gate modes
	"MODE_REST":   lua.LNumber(0),
	"MODE_SINGLE": lua.LNumber(1),
	"MODE_REPEAT": lua.LNumber(2),
	"MODE_HOLD":   lua.LNumber(3),

	// Sequencer patterns
	"MODE_SEQUENTIAL": lua.LNumber(0),
	"MODE_PINGPONG":   lua.LNumber(1),
	"MODE_RANDOM":     lua.LNumber(2),

	// Toggle modes
	"MODE_OFF": lua.LNumber(0),
	"MODE_ON":  lua.LNumber(1),

	// LP gate modes
	"MODE_LOWPASS":   lua.LNumber(0),
	"MODE_COMBO":     lua.LNumber(1),
	"MODE_AMPLITUDE": lua.LNumber(2),
}

func preloadSynth(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		fns := map[string]lua.LGFunction{}
		for _, name := range module.RegisteredTypes() {
			fns[name] = buildConstructor(name, mtx)
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

func buildConstructor(name string, mtx sync.Locker) func(state *lua.LState) int {
	return func(state *lua.LState) int {
		config := module.Config{}

		if state.GetTop() > 0 {
			table := state.CheckTable(1)
			fields := gluamapper.ToGoValue(table, mapperOpts).(map[interface{}]interface{})
			for k, v := range fields {
				switch asserted := v.(type) {
				case *lua.LUserData:
					switch ud := asserted.Value.(type) {
					case dsp.Valuer:
						config[k.(string)] = ud.Value()
					default:
						state.RaiseError("unconvertible userdata assigned: %T", ud)
					}
				default:
					config[k.(string)] = v
				}
			}
		}

		init, err := module.Lookup(name)
		if err != nil {
			state.RaiseError("%s", err.Error())
		}
		p, err := init(config)
		if err != nil {
			state.RaiseError("%s", err.Error())
		}

		table := addModuleMethods(state, p, mtx)
		state.Push(table)
		return 1
	}
}

func addModuleMethods(state *lua.LState, p module.Patcher, mtx sync.Locker) *lua.LTable {
	funcs := func(p module.Patcher) map[string]lua.LGFunction {
		fns := map[string]lua.LGFunction{
			// Methods lock and interact with the graph
			"close":     lock(moduleClose, mtx, p),
			"inputs":    lock(moduleInputs, mtx, p),
			"members":   lock(moduleMembers, mtx, p),
			"outputs":   lock(moduleOutputs, mtx, p),
			"reset":     lock(moduleReset, mtx, p),
			"resetOnly": lock(moduleResetOnly, mtx, p),
			"set":       lock(moduleSet, mtx, p),
			"state":     lock(moduleState, mtx, p),

			// Methods that don't need to lock the graph
			"ns":   moduleExtendNamespace(p),
			"out":  moduleOutput(p),
			"type": moduleType(p),
			"id":   moduleID(p),
		}

		for k, v := range moduleSpecificMethods(p, mtx) {
			if _, ok := fns[k]; ok {
				continue
			}
			fns[k] = v
		}

		return fns
	}(p)

	table := state.NewTable()
	state.RawSet(table, lua.LString("__namespace"), state.NewTable())
	state.RawSet(table, lua.LString("__type"), lua.LString("module"))
	state.SetFuncs(table, funcs)

	return table
}

type methodExposer interface {
	LuaMethods() map[string]module.LuaMethod
}

func moduleSpecificMethods(p module.Patcher, mtx sync.Locker) map[string]lua.LGFunction {
	var (
		luaMethods = map[string]lua.LGFunction{}
		methods    map[string]module.LuaMethod
	)

	if e, ok := p.(methodExposer); ok {
		methods = e.LuaMethods()
	}
	if methods == nil {
		return luaMethods
	}

	for k, v := range methods {
		switch fn := v.Func.(type) {
		case func(string) error:
			func(k string, lock bool, fn func(string) error) {
				luaMethods[k] = func(state *lua.LState) int {
					if lock {
						mtx.Lock()
						defer mtx.Unlock()
					}
					err := fn(state.CheckString(2))
					if err != nil {
						state.RaiseError(err.Error())
					}
					return 0
				}
			}(k, v.Lock, fn)
		case func() (string, error):
			func(k string, lock bool, fn func() (string, error)) {
				luaMethods[k] = func(state *lua.LState) int {
					if lock {
						mtx.Lock()
						defer mtx.Unlock()
					}
					r, err := fn()
					if err != nil {
						state.RaiseError(err.Error())
					}
					state.Push(lua.LString(r))
					return 1
				}
			}(k, v.Lock, fn)
		case func() (float64, error):
			func(k string, lock bool, fn func() (float64, error)) {
				luaMethods[k] = func(state *lua.LState) int {
					if lock {
						mtx.Lock()
						defer mtx.Unlock()
					}
					r, err := fn()
					if err != nil {
						state.RaiseError(err.Error())
					}
					state.Push(lua.LNumber(r))
					return 1
				}
			}(k, v.Lock, fn)
		}
	}
	return luaMethods
}

type lister interface {
	Inputs() map[string]*module.In
	Outputs() map[string]*module.Out
}

func moduleInputs(state *lua.LState, p module.Patcher) int {
	l, ok := p.(lister)
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
	l, ok := p.(lister)
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

type stateExposer interface {
	LuaState() map[string]interface{}
}

func moduleState(state *lua.LState, p module.Patcher) int {
	t := state.NewTable()
	if l, ok := p.(stateExposer); ok {
		for k, v := range l.LuaState() {
			t.RawSet(lua.LString(k), lua.LString(fmt.Sprintf("%v", v)))
		}
	}
	state.Push(t)
	return 1
}

type memberExposer interface {
	LuaMembers() []string
}

func moduleMembers(state *lua.LState, p module.Patcher) int {
	t := state.NewTable()
	if l, ok := p.(memberExposer); ok {
		for i, v := range l.LuaMembers() {
			t.Insert(i+1, lua.LString(v))
		}
	}
	state.Push(t)
	return 1
}

func moduleSet(state *lua.LState, p module.Patcher) int {
	self := state.CheckTable(1)
	inputs := map[interface{}]interface{}{}

	switch state.GetTop() {
	case 2:
		mapped := gluamapper.ToGoValue(state.CheckTable(2), mapperOpts)
		switch v := mapped.(type) {
		case map[interface{}]interface{}:
			inputs = v
		case []interface{}:
			for i, rv := range v {
				inputs[strconv.Itoa(i)] = rv
			}
		default:
			state.RaiseError("expected table, but got %T instead", mapped)
		}
	case 3:
		inputs[state.ToString(2)] = state.CheckAny(3)
	default:
		state.RaiseError("invalid number of arguments")
	}

	if err := setInputs(state, p, getNamespace(self), inputs); err != nil {
		state.RaiseError(err.Error())
	}
	state.Push(self)
	return 1
}

func setInputs(state *lua.LState, p module.Patcher, namespace []string, inputs map[interface{}]interface{}) error {
	for key, raw := range inputs {
		full := append(namespace, key.(string))

		if inputs, ok := raw.(map[interface{}]interface{}); ok {
			if err := setInputs(state, p, full, inputs); err != nil {
				return err
			}
			continue
		}

		name := strings.Join(full, "/")

		switch v := raw.(type) {
		case *lua.LUserData:
			switch mv := v.Value.(type) {
			case module.Patcher:
				if err := p.Patch(name, mv); err != nil {
					return err
				}
			case dsp.Valuer:
				if err := p.Patch(name, mv); err != nil {
					return err
				}
			default:
				state.RaiseError("unable to patch %T into %q", mv, name)
			}
		case lua.LNumber:
			if err := p.Patch(name, float64(v)); err != nil {
				return err
			}
		default:
			if err := p.Patch(name, raw); err != nil {
				return err
			}
		}
	}
	return nil
}

func moduleID(p module.Patcher) func(*lua.LState) int {
	return func(state *lua.LState) int {
		state.Push(lua.LString(p.ID()))
		return 1
	}
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
		state.RaiseError(err.Error())
	}
	return 0
}

func moduleClose(state *lua.LState, p module.Patcher) int {
	if err := p.Close(); err != nil {
		state.RaiseError(err.Error())
	}
	return 0
}

func moduleType(p module.Patcher) lua.LGFunction {
	return func(state *lua.LState) int {
		state.Push(lua.LString(p.Type()))
		return 1
	}
}

func moduleExtendNamespace(p module.Patcher) lua.LGFunction {
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

type lockingModuleMethod func(state *lua.LState, p module.Patcher) int

func lock(m lockingModuleMethod, mtx sync.Locker, p module.Patcher) lua.LGFunction {
	return func(state *lua.LState) int {
		mtx.Lock()
		defer mtx.Unlock()
		return m(state, p)
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
