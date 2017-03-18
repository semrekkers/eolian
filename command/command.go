// Package command provides the functionality of the executable
package command

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"runtime/trace"
	"syscall"
	"time"

	"buddin.us/eolian/engine"
	"buddin.us/eolian/lua"
	"buddin.us/eolian/module"        // Register standard modules
	_ "buddin.us/eolian/module/midi" // Register MIDI modules
	_ "buddin.us/eolian/module/osc"  // Register OSC modules
)

// Run is the main entrypoint for the eolian command
func Run(args []string) error {
	var (
		device             int
		seed               int64
		writeTrace, norepl bool
		frameSize          int
	)

	set := flag.NewFlagSet("eolian", flag.ContinueOnError)
	set.IntVar(&device, "output", 1, "output device")
	set.Int64Var(&seed, "seed", 0, "random seed")
	set.IntVar(&frameSize, "framesize", 256, "frame size")
	set.BoolVar(&writeTrace, "trace", false, "dump go trace tool information to trace.out")
	set.BoolVar(&norepl, "no-repl", false, "run without the REPL")
	if err := set.Parse(args); err != nil {
		return err
	}

	module.FrameSize = frameSize

	if writeTrace {
		f, err := os.OpenFile("trace.out", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := trace.Start(f); err != nil {
			return err
		}
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

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for s := range sig {
			switch s {
			case syscall.SIGUSR1:
				vm.DoString("Rack.patch()")
			case syscall.SIGUSR2:
				vm.DoString("Rack.build()")
			}
		}
	}()

	if norepl {
		waitForSignal()
	} else {
		if err := vm.REPL(); err != nil {
			return err
		}
	}
	return e.Close()
}

func waitForSignal() {
	sig := make(chan os.Signal)
	done := make(chan struct{})
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		done <- struct{}{}
	}()
	<-done
}
