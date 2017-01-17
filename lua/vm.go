// Package lua provides a Lua scripting layer
package lua

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/brettbuddin/eolian/module"
	"github.com/chzyer/readline"
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

const (
	historyFileVar     = "EOLIAN_HISTORY_FILE"
	defaultHistoryFile = "/tmp/eolian.tmp"
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
	openFilePath(state)
	state.PreloadModule("eolian.synth", preloadSynth(p))
	state.PreloadModule("eolian.synth.proxy", preloadSynthProxy)
	state.PreloadModule("eolian.theory", preloadTheory)

	state.SetGlobal("Engine", decoratePatcher(state, p))
	for k, fn := range globalFuncs {
		state.Register(k, fn)
	}
	if err := state.DoString(luaRack); err != nil {
		return nil, err
	}
	if err := state.DoString(luaUtil); err != nil {
		return nil, err
	}

	return &VM{state}, nil
}

func (vm *VM) LoadFile(path string) error {
	return vm.DoFile(path)
}

func (vm *VM) REPL() error {
	fmt.Println("Press Ctrl-D to exit")

	history := defaultHistoryFile
	if env := os.Getenv(historyFileVar); env != "" {
		history = env
	}

	l, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     history,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    readline.SegmentFunc(vm.completion),
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

func (vm *VM) completion(line [][]rune, pos int) [][]rune {
	if len(line) > 1 {
		return [][]rune{}
	}
	input := string(line[0])

	if start := strings.LastIndexAny(input, "({["); start > -1 && start != len(input)-1 {
		input = input[start+1:]
		pos -= start + 1
	}
	parts := strings.Split(input, ".")

	table := vm.GetGlobal("_G").(*lua.LTable)
	for _, part := range parts {
		table.ForEach(func(k, v lua.LValue) {
			if part == k.String() {
				if vt, ok := v.(*lua.LTable); ok {
					table = vt
				}
			}
		})
	}

	candidates := [][]rune{}
	table.ForEach(func(k, v lua.LValue) {
		c := k.String()
		if len(parts) > 1 {
			last := len(parts) - 1
			c = strings.Join(append(parts[:last], k.String()), ".")
			if strings.HasPrefix(parts[last], "__") {
				return
			}
		}
		candidates = append(candidates, []rune(c))
	})
	return candidates
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
