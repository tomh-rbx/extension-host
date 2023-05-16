package shutdown

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-host/exthost/common"
)

type Sysrq interface {
	Reboot() error
	Shutdown() error
}
type SysrqImpl struct{}

func NewSysrq() Sysrq {
	return &SysrqImpl{}
}

func (s SysrqImpl) Reboot() error {
	err := common.RunAsRoot("echo", "1", ">", "/proc/sys/kernel/sysrq")
	if err != nil {
		log.Err(err).Msg("Failed to set sysrq")
		return err
	}
	err = common.RunAsRoot("echo", "s", ">", "/proc/sysrq-trigger")
	if err != nil {
		log.Err(err).Msg("Failed to sync")
	}
	err = common.RunAsRoot("echo", "b", ">", "/proc/sysrq-trigger")
	return err
}

func (s SysrqImpl) Shutdown() error {
	err := common.RunAsRoot("echo", "1", ">", "/proc/sys/kernel/sysrq")
	if err != nil {
		log.Err(err).Msg("Failed to set sysrq")
		return err
	}
	err = common.RunAsRoot("echo", "s", ">", "/proc/sysrq-trigger")
	if err != nil {
		log.Err(err).Msg("Failed to sync")
	}
	err = common.RunAsRoot("echo", "o", ">", "/proc/sysrq-trigger")
	return err
}
