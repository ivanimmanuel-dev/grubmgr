package preview

import (
	"fmt"
	"os"
)

// firmware selects a distro OVMF pair for the private preview environment.
func firmware() (string, string, error) {
	for _, pair := range [][2]string{
		{"/usr/share/OVMF/OVMF_CODE_4M.fd", "/usr/share/OVMF/OVMF_VARS_4M.fd"},
		{"/usr/share/edk2/x64/OVMF_CODE.4m.fd", "/usr/share/edk2/x64/OVMF_VARS.4m.fd"},
	} {
		code, e1 := os.Stat(pair[0])
		vars, e2 := os.Stat(pair[1])
		if e1 == nil && e2 == nil && code.Mode().IsRegular() && vars.Mode().IsRegular() {
			return pair[0], pair[1], nil
		}
	}
	return "", "", fmt.Errorf("missing supported OVMF code/variable pair")
}
