package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Exit codes are part of the CLI protocol.
const (
	OK          = 0
	Usage       = 2
	Invalid     = 3
	Unsupported = 4
	Conflict    = 5
	IO          = 6
	Recovery    = 7
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Exit    int    `json:"-"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }
func Fail(exit int, code, format string, args ...any) error {
	return &Error{code, fmt.Sprintf(format, args...), exit}
}
func Write(w io.Writer, value any) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(value)
}
func Report(w io.Writer, err error, jsonMode bool) int {
	var e *Error
	if !errors.As(err, &e) {
		e = &Error{"IO_ERROR", err.Error(), IO}
	}
	if jsonMode {
		_ = Write(w, map[string]any{"error": e})
	} else {
		fmt.Fprintln(w, e.Error())
	}
	return e.Exit
}
