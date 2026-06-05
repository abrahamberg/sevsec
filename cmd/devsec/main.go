package main

import (
	"os"

	"github.com/abrahamberg/devsec/internal/cli/command"
)

func main() {

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

}
