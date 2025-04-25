//go:build freebsd
// +build freebsd

package xtoken

import "syscall"

func readPlatformMachineID() (string, error) {
	return syscall.Sysctl("kern.hostuuid")
}
