package main

import (
	"grubmgr/internal/debian"
	"os"
)

func main() {
	if len(os.Args) != 1 {
		os.Exit(2)
	}
	os.Exit(debian.Serve(os.Stdin, os.Stdout))
}
