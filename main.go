// Eolian is a modular synthesizer.
//
// Usage:
//
//   eolian [-device device_number] [-framesize buffer_size] [rack_file.lua]
//
package main

import (
	"fmt"
	"os"

	"buddin.us/eolian/command"
	"github.com/google/gops/agent"
)

func main() {
	if err := agent.Listen(nil); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err := command.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
