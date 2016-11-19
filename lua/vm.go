// Package lua provides a Lua scripting layer
package lua

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/brettbuddin/eolian/module"
	"github.com/chzyer/readline"
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

var (
	globalFuncs = map[string]lua.LGFunction{
		"hz":    hz,
		"ms":    ms,
		"pitch": pitch,
	}
	mapperOpts = gluamapper.Option{
		NameFunc: func(v string) string {
			return v
		},
	}
)

type VM struct {
	*lua.LState
}

func NewVM(p module.Patcher) (*VM, error) {
	state := lua.NewState()
	lua.OpenBase(state)
	lua.OpenDebug(state)
	lua.OpenString(state)
	OpenFilePath(state)
	OpenSynth(state, p)
	OpenTheory(state)

	// Add go functions
	for k, fn := range globalFuncs {
		state.Register(k, fn)
	}

	// Add rack behavior
	if err := state.DoString(luaRack); err != nil {
		return nil, err
	}

	return &VM{state}, nil
}

func (vm *VM) LoadFile(path string) error {
	return vm.DoFile(path)
}

func (vm *VM) REPL() error {
	fmt.Println("Press Ctrl-D to exit")
	l, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     "/tmp/carrier.tmp",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer l.Close()
	log.SetOutput(l.Stderr())

	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if line == "exit" {
			break
		}
		if err := vm.DoString(line); err != nil {
			log.Println("error:", err)
		}
	}
	return nil
}

func hz(state *lua.LState) int {
	value := state.ToNumber(1)
	hz := module.Frequency(float64(value))
	state.Push(&lua.LUserData{Value: hz})
	return 1
}

func pitch(state *lua.LState) int {
	value := state.ToString(1)
	pitch, err := module.ParsePitch(value)
	if err != nil {
		state.RaiseError("%s", err.Error())
	}
	state.Push(&lua.LUserData{Value: pitch})
	return 1
}

func ms(state *lua.LState) int {
	value := state.ToNumber(1)
	ms := module.Duration(float64(value))
	state.Push(&lua.LUserData{Value: ms})
	return 1
}
