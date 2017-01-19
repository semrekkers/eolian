package lua

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

type session struct {
	sync.Mutex
	history []string
}

func newSession(state *lua.LState) *session {
	s := &session{
		history: []string{},
	}
	fns := map[string]lua.LGFunction{
		"reset": s.reset,
		"save":  s.save,
	}
	state.RegisterModule("session", fns)
	return s
}

func (s *session) addLine(l string) {
	s.Lock()
	s.history = append(s.history, l)
	s.Unlock()
}

func (s *session) save(state *lua.LState) int {
	s.Lock()
	defer s.Unlock()

	name := state.CheckString(1)
	abs, err := filepath.Abs(name)
	if err != nil {
		state.RaiseError("absolute path %s: %s", name, err)
	}
	f, err := os.OpenFile(abs, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		state.RaiseError("open %s: %s", abs, err)
	}
	defer f.Close()

	for _, l := range s.history[:len(s.history)-1] {
		f.WriteString(l + "\n")
	}
	fmt.Printf("Session written to %s\n", abs)
	return 0
}

func (s *session) reset(state *lua.LState) int {
	s.Lock()
	s.history = []string{}
	s.Unlock()
	return 0
}
