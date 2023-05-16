//go:build !linux

package shutdown

import (
	"fmt"
	"syscall"
)

type Syscall interface {
  Reboot() error
  Shutdown() error
}
type SyscallImpl struct{}

func NewSyscall() Syscall {
  return &SyscallImpl{}
}

func (s SyscallImpl) Reboot() error {
	_, _, err := syscall.Syscall(syscall.SYS_REBOOT, 0, 0, 0)
	if err != 0 {
		return fmt.Errorf("error rebooting: %v", err)
	}
	return nil
}

func (s SyscallImpl) Shutdown() error {
	_, _, err := syscall.Syscall(syscall.SYS_SHUTDOWN, 0, 0, 0)
	if err != 0 {
		return fmt.Errorf("error shutting down: %v", err)
	}
	return nil
}
