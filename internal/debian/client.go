package debian

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"grubmgr/internal/output"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func Available() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := os.Stat(Helper); err != nil {
		return false
	}
	return true
}

// Call sends a typed request to the installed helper through Polkit.
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
	interrupted, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithTimeout(interrupted, 5*time.Minute)
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
		if ctx.Err() == context.Canceled {
			return output.Fail(output.Usage, "CANCELLED", "operation cancelled")
		}
		if exit, ok := runErr.(*exec.ExitError); ok && exit.ExitCode() == 126 {
			return output.Fail(output.Usage, "AUTH_CANCELLED", "administrator authentication cancelled")
		} else if ok && exit.ExitCode() == 127 {
			return output.Fail(output.Unsupported, "AUTH_REQUIRED", "administrator authentication denied; use an active local login with an administrator account")
		}
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
