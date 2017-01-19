// Package lua provides a Lua scripting layer
package lua

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/chzyer/readline"
	lua "github.com/yuin/gopher-lua"

	"github.com/brettbuddin/eolian/module"
)

const (
	historyFileVar     = "EOLIAN_HISTORY_FILE"
	defaultHistoryFile = "/tmp/eolian.tmp"
)

type VM struct {
	*lua.LState
}

func NewVM(p module.Patcher, mtx *sync.Mutex) (*VM, error) {
	state := lua.NewState()
	lua.OpenBase(state)
	lua.OpenDebug(state)
	lua.OpenString(state)

	openFilePath(state)

	state.PreloadModule("eolian.synth", preloadSynth(mtx))
	state.PreloadModule("eolian.synth.proxy", preloadSynthProxy)
	state.PreloadModule("eolian.theory", preloadTheory)

	state.SetGlobal("Engine", decoratePatcher(state, p, mtx))
	for k, fn := range valueFuncs {
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

	session := newSession(vm.LState)

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
		if line == "exit" || line == "quit" {
			break
		}
		session.addLine(line)
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
