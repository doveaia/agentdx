//go:build linux || freebsd || openbsd || netbsd

package session

import "syscall"

// getSysProcAttr returns platform-specific process attributes for daemon management.
// On Unix-like systems (Linux, BSD), Setpgid creates a new process group.
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
