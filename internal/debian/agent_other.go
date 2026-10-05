//go:build !linux

package debian

import "context"

func terminalAgent(context.Context) (func(), error) { return func() {}, nil }
