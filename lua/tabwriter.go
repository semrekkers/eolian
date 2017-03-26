package lua

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	lua "github.com/yuin/gopher-lua"
)

func preloadTabWriter(state *lua.LState) int {
	mod := state.NewTable()
	state.SetFuncs(mod, map[string]lua.LGFunction{
		"new": newTabWriter,
	})
	state.Push(mod)
	return 1
}

func newTabWriter(state *lua.LState) int {
	minwidth := state.CheckInt(1)
	tabwidth := state.CheckInt(2)
	padwidth := state.CheckInt(3)
	pad := state.CheckString(4)
	align := state.OptString(5, "")

	var flags uint
	if align == "alignRight" {
		flags = tabwriter.AlignRight
	}

	buf := bytes.NewBuffer(nil)
	w := tabwriter.NewWriter(buf, minwidth, tabwidth, padwidth, pad[0], flags)

	t := state.NewTable()
	state.SetFuncs(t, map[string]lua.LGFunction{
		"write": func(state *lua.LState) int {
			str := state.CheckString(1)
			fmt.Fprint(w, str)
			return 0
		},
		"flush": func(state *lua.LState) int {
			w.Flush()
			state.Push(lua.LString(buf.String()))
			return 1
		},
	})
	state.Push(t)
	return 1
}
