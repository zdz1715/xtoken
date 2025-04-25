//go:build openbsd
// +build openbsd

package xtoken

import "syscall"

func readPlatformMachineID() (string, error) {
	return syscall.Sysctl("hw.uuid")
}
