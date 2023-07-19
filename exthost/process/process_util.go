package stopprocess

import (
	"fmt"
	"github.com/mitchellh/go-ps"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-host/exthost/common"
	"github.com/steadybit/extension-kit/extutil"
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
		if process, err := ps.FindProcess(p); err == nil {
			log.Info().Int("pid", p).Str("name", process.Executable()).Msg("Stopping process")
		} else {
			continue
		}

		var err error
		if runtime.GOOS == "windows" {
			err = stopProcessWindows(p, force)
		} else {
			err = stopProcessUnix(p, force)
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

func stopProcessWindows(pid int, force bool) error {
	if force {
		err := exec.Command("taskkill", "/f", "/pid", fmt.Sprintf("%d", pid)).Run()
		if err != nil {
			log.Error().Err(err).Int("pid", pid).Msg("Failed to kill process")
		}
		return err
	}

	err := exec.Command("taskkill", "/pid", fmt.Sprintf("%d", pid)).Run()
	if err != nil {
		log.Error().Err(err).Int("pid", pid).Msg("Failed to send SIGTERM")
	}
	return err
}

func stopProcessUnix(pid int, force bool) error {
	if force {
		err := syscall.Kill(pid, syscall.SIGKILL)
		if err != nil {
			log.Error().Err(err).Int("pid", pid).Msg("Failed to kill process via syscall")
			err = common.RunAsRoot("kill", "-9", fmt.Sprintf("%d", pid))
		}
		if err != nil {
			log.Error().Err(err).Int("pid", pid).Msg("Failed to kill process via exec")
		}
		return err
	}

	err := syscall.Kill(pid, syscall.SIGTERM)
	if err != nil {
		log.Error().Err(err).Int("pid", pid).Msg("Failed to send SIGTERM via syscall")
		err = common.RunAsRoot("kill", fmt.Sprintf("%d", pid))
	}
	if err != nil {
		log.Error().Err(err).Int("pid", pid).Msg("Failed to send SIGTERM via exec")
	}
	return err
}

func FindProcessIds(processOrPid string) []int {
	pid := extutil.ToInt(processOrPid)
	if pid > 0 {
		return []int{pid}
	}

	var pids []int
	processes, err := ps.Processes()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list processes")
		return nil
	}
	for _, process := range processes {
		if strings.Contains(strings.TrimSpace(process.Executable()), processOrPid) {
			pids = append(pids, process.Pid())
		}
	}
	return pids
}
