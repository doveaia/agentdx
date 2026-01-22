//go:build darwin

package session

import "syscall"

// getSysProcAttr returns platform-specific process attributes for daemon management.
// On macOS (Darwin), Setpgid is not available, so we return nil.
// The process will still be detached from the parent's terminal.
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
