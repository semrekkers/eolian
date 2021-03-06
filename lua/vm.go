// Package lua provides a Lua scripting layer
package lua

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/chzyer/readline"
	lua "github.com/yuin/gopher-lua"

	"buddin.us/eolian/lua/synth"
	"buddin.us/eolian/lua/theory"
	"buddin.us/eolian/module"
)

const (
	historyFileVar     = "EOLIAN_HISTORY_FILE"
	defaultHistoryFile = "/tmp/eolian"
)

// VM is the Lua virtual machine
type VM struct {
	*lua.LState
}

// NewVM returns a new lua virtual machine centered around a Patcher
func NewVM(p module.Patcher, mtx sync.Locker) (*VM, error) {
	state := lua.NewState()
	lua.OpenBase(state)
	lua.OpenDebug(state)
	lua.OpenString(state)

	state.PreloadModule("eolian.filepath", preloadFilepath)
	state.PreloadModule("eolian.repl", preloadLibFile("lua/lib/repl.lua"))
	state.PreloadModule("eolian.runtime", preloadRuntime)
	state.PreloadModule("eolian.sort", preloadSort)
	state.PreloadModule("eolian.string", preloadString)
	state.PreloadModule("eolian.synth", synth.Preload(mtx))
	state.PreloadModule("eolian.func", preloadLibFile("lua/lib/func.lua"))
	state.PreloadModule("eolian.synth.control", preloadLibFile("lua/lib/synth/control.lua"))
	state.PreloadModule("eolian.synth.proxy", preloadSynthProxy)
	state.PreloadModule("eolian.rack.route", preloadLibFile("lua/lib/rack/route.lua"))
	state.PreloadModule("eolian.rack.mount", preloadLibFile("lua/lib/rack/mount.lua"))
	state.PreloadModule("eolian.tabwriter", preloadTabWriter)
	state.PreloadModule("eolian.theory", theory.Preload)
	state.PreloadModule("eolian.time", preloadTime)
	state.PreloadModule("eolian.value", preloadValue)

	state.SetGlobal("Engine", synth.CreateModule(state, p, mtx))
	if err := loadLibFile(state, "lua/lib/rack/rack.lua"); err != nil {
		return nil, err
	}
	if err := state.DoString("repl = require('eolian.repl')"); err != nil {
		return nil, err
	}
	if err := state.DoString("for k,v in pairs(require('eolian.value')) do _G[k] = v end"); err != nil {
		return nil, err
	}
	return &VM{state}, nil
}

// REPL starts the read-eval-print-loop
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
		EOFPrompt:       "See You Space Cowboy...",
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
		if line == "" {
			continue
		}

		if err := vm.DoString(fmt.Sprintf("repl.exec(%q)", line)); err != nil {
			fmt.Println("error:", err)
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
	}
	parts := strings.Split(input, ".")

	table := vm.GetGlobal("_G").(*lua.LTable)
	for i, part := range parts {
		var found bool
		table.ForEach(func(k, v lua.LValue) {
			if part == k.String() {
				if vt, ok := v.(*lua.LTable); ok {
					table = vt
					found = true
					return
				}
			}
		})
		if !found && i < len(parts)-1 {
			return [][]rune{}
		}
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
	sort.Slice(candidates, func(i, j int) bool {
		return string(candidates[i]) < string(candidates[j])
	})
	return candidates
}
