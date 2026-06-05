package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		fmt.Println("Usage: devsec [options]")
	} else {
		fmt.Println("Running DevSec with options:", os.Args[1:])
	}
}
