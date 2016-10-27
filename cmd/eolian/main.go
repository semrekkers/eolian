package main

import (
	"log"
	"os"

	"github.com/brettbuddin/eolian/command"
)

func main() {
	if err := command.Run(os.Args[1:]); err != nil {
		log.Println(err)
	}
}
