package synth

import (
	"sync"

	"buddin.us/eolian/dsp"
	"buddin.us/eolian/module"
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

const (
	patchStateKey = "__patchstate"
	namespaceKey  = "__namespace"
	patcherKey    = "__patcher"
)

func constructor(name string, mtx sync.Locker) func(state *lua.LState) int {
	return func(state *lua.LState) int {
		config := getConfig(state)
		init, err := module.Lookup(name)
		if err != nil {
			state.RaiseError("%s", err.Error())
		}
		p, err := init(config)
		if err != nil {
			state.RaiseError("%s", err.Error())
		}
		state.Push(CreateModule(state, p, mtx))
		return 1
	}
}

func getConfig(state *lua.LState) module.Config {
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

	return config
}

func CreateModule(state *lua.LState, p module.Patcher, mtx sync.Locker) *lua.LTable {
	data := state.NewUserData()
	data.Value = p

	table := state.NewTable()
	state.RawSet(table, lua.LString(patchStateKey), state.NewTable())
	state.RawSet(table, lua.LString(namespaceKey), state.NewTable())
	state.RawSet(table, lua.LString(patcherKey), data)
	state.SetFuncs(table, moduleMethods(p, mtx))

	return table
}

func moduleMethods(p module.Patcher, mtx sync.Locker) map[string]lua.LGFunction {
	fns := map[string]lua.LGFunction{
		"close":       patcherClose(mtx),
		"finishPatch": patcherFinishPatch(mtx),
		"id":          patcherID(mtx),
		"inputs":      patcherInputs(mtx),
		"ns":          patcherExtendNamespace(mtx),
		"members":     patcherMembers(mtx),
		"out":         patcherOutput(mtx),
		"outputs":     patcherOutputs(mtx),
		"reset":       patcherReset(mtx),
		"resetOnly":   patcherResetOnly(mtx),
		"set":         patcherSet(mtx),
		"startPatch":  patcherStartPatch(mtx),
		"state":       patcherState(mtx),
		"type":        patcherType(mtx),
	}

	for k, v := range exposedMethods(p, mtx) {
		if _, ok := fns[k]; ok {
			continue
		}
		fns[k] = v
	}

	return fns
}
