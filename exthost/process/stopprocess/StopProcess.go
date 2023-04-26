package stopprocess

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

func StopProcesses(pid []int, force bool) error {
	if len(pid) == 0 {
		log.Error().Msg("No pid to stop")
		return nil
	}
	errors := make([]string, 0)
	for _, p := range pid {
		var err error
		if runtime.GOOS == "windows" {
			err = StopProcessWindows(p, force)
		} else {
			err = StopProcessUnix(p, force)
		}
		if err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("fail to stop processes : %s", strings.Join(errors, ", "))
	}
	return nil
}

func StopProcessWindows(pid int, force bool) error {
	if force {
		err := exec.Command("taskkill", "/f", "/pid", fmt.Sprintf("%d", pid)).Run()
		if err != nil {
			log.Error().Err(err).Int("pid", pid).Msg("Failed to kill process")
			return err
		}
	} else {
		err := exec.Command("taskkill", "/pid", fmt.Sprintf("%d", pid)).Run()
		if err != nil {
			log.Error().Err(err).Int("pid", pid).Msg("Failed to send SIGTERM")
			return err
		}
	}
	return nil
}

func StopProcessUnix(pid int, force bool) error {
	if force {
		err := syscall.Kill(pid, syscall.SIGKILL)
		if err != nil {
			log.Error().Err(err).Int("pid", pid).Msg("Failed to kill process")
			return err
		}
	} else {
		err := syscall.Kill(pid, syscall.SIGTERM)
		if err != nil {
			log.Error().Err(err).Int("pid", pid).Msg("Failed to send SIGTERM")
			return err
		}
	}
	return nil
}
