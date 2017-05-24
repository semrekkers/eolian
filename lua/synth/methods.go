package synth

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"

	"buddin.us/eolian/module"
)

var mapperOpts = gluamapper.Option{
	NameFunc: func(v string) string {
		return v
	},
}

func getPatcher(state *lua.LState, t *lua.LTable) (module.Patcher, error) {
	v := state.GetField(t, patcherKey)
	if data, ok := v.(*lua.LUserData); ok {
		if p, ok := data.Value.(module.Patcher); ok {
			return p, nil
		}
		return nil, fmt.Errorf("expected module.Patcher at %s; got %T", patcherKey, data.Value)
	}
	return nil, fmt.Errorf("expected userdata at %s; got %T", patcherKey, v)
}

func patcherID(_ sync.Locker) func(*lua.LState) int {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}
		state.Push(lua.LString(p.ID()))
		return 1
	}
}

func patcherType(_ sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}
		state.Push(lua.LString(p.ID()))
		return 1
	}
}

func patcherInputs(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}

		namespace := strings.Join(getNamespace(state, self), "/")
		t := state.NewTable()
		mtx.Lock()
		for k, v := range p.Inputs() {
			if !strings.HasPrefix(k, namespace) {
				continue
			}
			t.RawSet(lua.LString(k), lua.LString(v.SourceName()))
		}
		mtx.Unlock()
		state.Push(t)
		return 1
	}
}

func patcherOutputs(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}

		namespace := strings.Join(getNamespace(state, self), "/")
		t := state.NewTable()
		mtx.Lock()
		for k, v := range p.Outputs() {
			if !strings.HasPrefix(k, namespace) {
				continue
			}
			dests := state.NewTable()
			for i, name := range v.DestinationNames() {
				dests.Insert(i+1, lua.LString(name))
			}
			t.RawSet(lua.LString(k), dests)
		}
		mtx.Unlock()
		state.Push(t)
		return 1
	}
}

func patcherExtendNamespace(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)

		next := state.NewTable()
		if current := state.GetField(self, namespaceKey).(*lua.LTable); current != nil {
			current.ForEach(func(_, v lua.LValue) { next.Append(v) })
		}

		name := strings.Trim(state.ToString(2), "/")
		next.Append(lua.LString(name))

		t := state.NewTable()
		t.RawSetString(namespaceKey, next)

		mt := state.NewTable()
		mt.RawSetString("__index", self)

		state.SetMetatable(t, mt)
		state.Push(t)
		return 1
	}
}

type stateExposer interface {
	LuaState() map[string]interface{}
}

func patcherState(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}

		t := state.NewTable()
		if l, ok := p.(stateExposer); ok {
			mtx.Lock()
			for k, v := range l.LuaState() {
				t.RawSet(lua.LString(k), lua.LString(fmt.Sprintf("%v", v)))
			}
			mtx.Unlock()
		}
		state.Push(t)
		return 1
	}
}

type memberExposer interface {
	LuaMembers() []string
}

func patcherMembers(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}

		t := state.NewTable()
		if l, ok := p.(memberExposer); ok {
			mtx.Lock()
			for i, v := range l.LuaMembers() {
				t.Insert(i+1, lua.LString(v))
			}
			mtx.Unlock()
		}
		state.Push(t)
		return 1
	}
}

func patcherSet(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}

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

		namespace := getNamespace(state, self)

		mtx.Lock()
		err = setInputs(state, self, p, namespace, inputs)
		mtx.Unlock()
		if err != nil {
			state.RaiseError(err.Error())
		}

		state.Push(self)
		return 1
	}
}

func getNamespace(state *lua.LState, table *lua.LTable) []string {
	raw := state.GetField(table, namespaceKey).(*lua.LTable)
	namespace := gluamapper.ToGoValue(raw, mapperOpts)

	segs := []string{}
	if ns, ok := namespace.([]interface{}); ok {
		for _, v := range ns {
			segs = append(segs, v.(string))
		}
	}
	return segs
}

func setInputs(state *lua.LState, self *lua.LTable, p module.Patcher, namespace []string, inputs map[interface{}]interface{}) error {
	for key, raw := range inputs {
		full := append(namespace, key.(string))
		if inputs, ok := raw.(map[interface{}]interface{}); ok {
			if err := setInputs(state, self, p, full, inputs); err != nil {
				return err
			}
			continue
		}
		name := strings.Join(full, "/")

		// Mark this input as being touched
		patchState := state.GetField(self, patchStateKey).(*lua.LTable)
		patchState.RawSetString(name, lua.LBool(true))
		state.SetField(self, patchStateKey, patchState)

		switch v := raw.(type) {
		case *lua.LUserData:
			if err := p.Patch(name, v.Value); err != nil {
				return err
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

func patcherClose(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}

		mtx.Lock()
		err = p.Close()
		mtx.Unlock()
		if err != nil {
			state.RaiseError(err.Error())
		}
		return 0
	}
}

func patcherResetOnly(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}

		names := state.CheckTable(1)
		inputs := []string{}
		names.ForEach(func(k, v lua.LValue) {
			inputs = append(inputs, v.String())
		})

		mtx.Lock()
		err = p.ResetOnly(inputs)
		mtx.Unlock()
		if err != nil {
			state.RaiseError("%s", err.Error())
		}
		return 0
	}
}

func patcherReset(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}

		mtx.Lock()
		err = p.Reset()
		mtx.Unlock()
		if err != nil {
			state.RaiseError(err.Error())
		}
		return 0
	}
}

func patcherOutput(_ sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}

		var names []string
		if top := state.GetTop(); top > 1 {
			for i := 0; i <= top; i++ {
				v := state.Get(i + 2)
				if v == lua.LNil {
					continue
				}
				names = append(names, v.String())
			}
		}
		if len(names) == 0 {
			names = append(names, "output")
		}

		namespace := getNamespace(state, self)
		for _, n := range names {
			state.Push(&lua.LUserData{
				Value: module.Port{p, strings.Join(append(namespace, n), "/")},
			})
		}
		return len(names)
	}
}

func patcherStartPatch(_ sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		state.SetField(self, patchStateKey, state.NewTable())
		return 0
	}
}

func patcherFinishPatch(mtx sync.Locker) lua.LGFunction {
	return func(state *lua.LState) int {
		self := state.CheckTable(1)
		p, err := getPatcher(state, self)
		if err != nil {
			state.RaiseError(err.Error())
		}

		exclusions := state.OptTable(2, state.NewTable())

		patchState := state.GetField(self, patchStateKey).(*lua.LTable)
		mtx.Lock()
		inputs := p.Inputs()
		mtx.Unlock()

		for _, in := range inputs {
			var found bool
			patchState.ForEach(func(k, _ lua.LValue) {
				if in.Name == k.String() {
					found = true
					return
				}
			})
			if !found {
				var excluded bool
				exclusions.ForEach(func(_, v lua.LValue) {
					if in.Name == v.String() {
						excluded = true
					}
				})
				if excluded {
					continue
				}

				mtx.Lock()
				err := in.Close()
				mtx.Unlock()
				if err != nil {
					state.RaiseError(err.Error())
				}
			}
		}
		return 0
	}
}

type methodExposer interface {
	LuaMethods() map[string]module.LuaMethod
}

func exposedMethods(p module.Patcher, mtx sync.Locker) map[string]lua.LGFunction {
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
