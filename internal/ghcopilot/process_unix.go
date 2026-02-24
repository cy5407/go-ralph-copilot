//go:build !windows

package ghcopilot

import "os/exec"

// setSysProcAttr on non-Windows: no-op
func setSysProcAttr(cmd *exec.Cmd) {}

// killProcessTree on non-Windows: no-op (context cancel handles it)
func killProcessTree(pid int) {}
