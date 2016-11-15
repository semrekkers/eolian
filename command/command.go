package command

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/brettbuddin/eolian/engine"
	"github.com/brettbuddin/eolian/lua"
	_ "github.com/brettbuddin/eolian/midi"
	_ "github.com/brettbuddin/eolian/osc"
)

func Run(args []string) error {
	var (
		device int
		seed   int64
	)

	set := flag.NewFlagSet("eolian", flag.ContinueOnError)
	set.IntVar(&device, "output", 1, "output device")
	set.Int64Var(&seed, "seed", 0, "random seed")
	if err := set.Parse(args); err != nil {
		return err
	}

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

	vm, err := lua.NewVM(e)
	if err != nil {
		return err
	}

	if len(os.Args) > 1 {
		vm.DoString(fmt.Sprintf("Rack.load('%s')", os.Args[1]))
	}

	return vm.REPL()
}
