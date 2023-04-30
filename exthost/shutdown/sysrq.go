package shutdown

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-host/exthost/common"
)

func RebootSysrq() error {
	//echo 1 > /proc/sys/kernel/sysrq
	err := common.RunAsRoot("echo", "1", ">", "/proc/sys/kernel/sysrq")
	if err != nil {
		log.Err(err).Msg("Failed to set sysrq")
		return err
	}
	// echo s > /proc/sysrq-trigger
	err = common.RunAsRoot("echo", "s", ">", "/proc/sysrq-trigger")
	if err != nil {
		log.Err(err).Msg("Failed to sync")
	}
	//echo b > /proc/sysrq-trigger
	err = common.RunAsRoot("echo", "b", ">", "/proc/sysrq-trigger")
	return err
}

func ShutdownSysrq() error {
	//echo 1 > /proc/sys/kernel/sysrq
	err := common.RunAsRoot("echo", "1", ">", "/proc/sys/kernel/sysrq")
	if err != nil {
		log.Err(err).Msg("Failed to set sysrq")
		return err
	}
	// echo s > /proc/sysrq-trigger
	err = common.RunAsRoot("echo", "s", ">", "/proc/sysrq-trigger")
	if err != nil {
		log.Err(err).Msg("Failed to sync")
	}
	//echo b > /proc/sysrq-trigger
	err = common.RunAsRoot("echo", "o", ">", "/proc/sysrq-trigger")
	return err
}
