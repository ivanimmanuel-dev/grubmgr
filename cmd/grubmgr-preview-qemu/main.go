package main

import (
	"fmt"
	"grubmgr/internal/preview"
	"os"
)

func main() {
	if err := preview.QEMU(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
