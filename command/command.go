// Package command provides the functionality of the executable
package command

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime/trace"
	"time"

	"github.com/brettbuddin/eolian/engine"
	"github.com/brettbuddin/eolian/lua"
	_ "github.com/brettbuddin/eolian/midi" // Register MIDI modules
	_ "github.com/brettbuddin/eolian/osc"  // Register OSC modules
)

func Run(args []string) error {
	var (
		device     int
		seed       int64
		writeTrace bool
	)

	set := flag.NewFlagSet("eolian", flag.ContinueOnError)
	set.IntVar(&device, "output", 1, "output device")
	set.Int64Var(&seed, "seed", 0, "random seed")
	set.BoolVar(&writeTrace, "trace", false, "dump go trace tool information to trace.out")
	if err := set.Parse(args); err != nil {
		return err
	}

	if writeTrace {
		f, err := os.OpenFile("trace.out", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
		if err != nil {
			return err
		}
		defer f.Close()
		trace.Start(f)
		defer trace.Stop()
	}

	fmt.Println("PID:", os.Getpid())

	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	fmt.Println("Seed:", seed)
	rand.Seed(seed)

	e, err := engine.New(device)
	if err != nil {
		return err
	}
	go e.Run()
	go func() {
		for err := range e.Errors() {
			fmt.Println("engine error:", err)
		}
	}()

	vm, err := lua.NewVM(e, &e.Mutex)
	if err != nil {
		return err
	}

	if len(set.Args()) > 0 {
		if err := vm.DoString(fmt.Sprintf("Rack.load('%s')", set.Arg(0))); err != nil {
			return err
		}
	}

	return vm.REPL()
}
