//go:build windows

package session

import "syscall"

// getSysProcAttr returns platform-specific process attributes for daemon management.
// On Windows, we use CREATE_NEW_PROCESS_GROUP to detach from the parent console.
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
