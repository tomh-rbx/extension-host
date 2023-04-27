package shutdown

import (
	"github.com/rs/zerolog/log"
	"os/exec"
)

func RebootSysrq() error {
	//echo 1 > /proc/sys/kernel/sysrq
	err := exec.Command("echo", "1", ">", "/proc/sys/kernel/sysrq").Run()
	if err != nil {
		log.Err(err).Msg("Failed to set sysrq")
		return err
	}
	// echo s > /proc/sysrq-trigger
	err = exec.Command("echo", "s", ">", "/proc/sysrq-trigger").Run()
	if err != nil {
		log.Err(err).Msg("Failed to sync")
	}
	//echo b > /proc/sysrq-trigger
	err = exec.Command("echo", "b", ">", "/proc/sysrq-trigger").Run()
	return err
}

func ShutdownSysrq() error {
	//echo 1 > /proc/sys/kernel/sysrq
	err := exec.Command("echo", "1", ">", "/proc/sys/kernel/sysrq").Run()
	if err != nil {
		log.Err(err).Msg("Failed to set sysrq")
		return err
	}
	// echo s > /proc/sysrq-trigger
	err = exec.Command("echo", "s", ">", "/proc/sysrq-trigger").Run()
	if err != nil {
		log.Err(err).Msg("Failed to sync")
	}
	//echo b > /proc/sysrq-trigger
	err = exec.Command("echo", "o", ">", "/proc/sysrq-trigger").Run()
	return err
}
