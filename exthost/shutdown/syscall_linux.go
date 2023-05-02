//go:build linux

package shutdown

import (
	"github.com/rs/zerolog/log"
	"syscall"
)

func RebootSyscall() error {
	log.Info().Msg("Rebooting via syscall `syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)`")
	err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	if err != nil {
		log.Err(err).Msg("Failed to reboot")
		return err
	}
	return nil
}

func ShutdownSyscall() error {
	err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	if err != nil {
		log.Error().Err(err).Msg("Failed to shutdown")
		return err
	}
	return nil
}
