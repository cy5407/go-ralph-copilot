//go:build windows

package ghcopilot

import (
	"fmt"
	"os/exec"
	"syscall"
)

// setSysProcAttr 在 Windows 上設定 process group，讓 kill 可以殺整個 process tree
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// killProcessTree 在 Windows 上強制殺掉整個 process tree（包含子進程）
func killProcessTree(pid int) {
	// taskkill /F /T /PID 殺整個 process tree
	kill := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	_ = kill.Run()
}
