//go:build linux

package shutdown

import (
	"github.com/rs/zerolog/log"
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
	log.Info().Msg("Rebooting via syscall `syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)`")
	err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	if err != nil {
		log.Err(err).Msg("Failed to reboot")
		return err
	}
	return nil
}

func (s SyscallImpl) Shutdown() error {
	err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	if err != nil {
		log.Error().Err(err).Msg("Failed to shutdown")
		return err
	}
	return nil
}
