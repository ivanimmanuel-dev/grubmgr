package debian

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"grubmgr/internal/output"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func Available() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := os.Stat(Helper); err != nil {
		return false
	}
	marker, err := os.ReadFile("/etc/grubmgr/vm-test")
	if err != nil || string(marker) != VMMarker {
		return false
	}
	dmi, err := os.ReadFile("/sys/class/dmi/id/product_name")
	return err == nil && strings.TrimSpace(string(dmi)) == "grubmgr-disposable-v1"
}

// Call invokes only the installed helper; neither path is configurable by a theme.
func Call(req Request, result any) error {
	if runtime.GOOS != "linux" || os.Geteuid() == 0 {
		return output.Fail(output.Unsupported, "UNPRIVILEGED_REQUIRED", "run the CLI as an ordinary Linux user")
	}
	if err := secure(Helper, false); err != nil {
		return err
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if len(data) > MaxRequest {
		return fmt.Errorf("request exceeds limit")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	stopAgent, err := terminalAgent(ctx)
	if err != nil {
		return err
	}
	defer stopAgent()
	cmd := exec.CommandContext(ctx, "/usr/bin/pkexec", Helper)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	runErr := cmd.Run()
	var response Response
	if err = json.Unmarshal(out.Bytes(), &response); err != nil {
		return fmt.Errorf("helper did not return a response: %v", runErr)
	}
	if response.Error != "" {
		return output.Fail(response.Code, "HELPER_REFUSED", "%s", response.Error)
	}
	if runErr != nil {
		return runErr
	}
	return json.Unmarshal(response.Result, result)
}
