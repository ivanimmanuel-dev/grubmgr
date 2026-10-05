//go:build windows

package fsx

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
)

func SingleLink(f *os.File) error {
	var i windows.ByHandleFileInformation
	if e := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &i); e != nil {
		return e
	}
	if i.NumberOfLinks != 1 {
		return fmt.Errorf("multiply linked file refused")
	}
	return nil
}
