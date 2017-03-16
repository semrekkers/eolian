package main

import (
	"log"
	"os"

	"github.com/brettbuddin/eolian/command"
	"github.com/google/gops/agent"
)

func main() {
	if err := agent.Listen(nil); err != nil {
		log.Fatal(err)
	}
	if err := command.Run(os.Args[1:]); err != nil {
		log.Println(err)
	}
}
