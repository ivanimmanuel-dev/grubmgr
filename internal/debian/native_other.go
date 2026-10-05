//go:build !linux

package debian

import (
	"fmt"
	"os"
)

func secure(string, bool) error { return fmt.Errorf("activation helper requires Linux") }
func atomicReplace(*os.Root, string, []byte) error {
	return fmt.Errorf("activation helper requires Linux")
}
func acquire(string) (func(), error)  { return nil, fmt.Errorf("activation helper requires Linux") }
func runTool(string, ...string) error { return fmt.Errorf("activation helper requires Linux") }
func helperIdentity() error           { return fmt.Errorf("activation helper requires Linux") }
func syncTree(string) error           { return fmt.Errorf("activation helper requires Linux") }
