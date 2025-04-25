//go:build !darwin && !linux && !freebsd && !openbsd && !windows
// +build !darwin,!linux,!freebsd,!openbsd,!windows

package xtoken

import "errors"

func readPlatformMachineID() (string, error) {
	return "", errors.New("not implemented")
}
