package shutdown

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-host/exthost/common"
)

func RebootSysrq() error {
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

func ShutdownSysrq() error {
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
