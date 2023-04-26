package shutdown

import (
	"github.com/rs/zerolog/log"
	"os/exec"
	"runtime"
)

func getShutdownCommand() []string {

	if runtime.GOOS == "windows" {
		return []string{"shutdown.exe", "/s", "/t", "0"}
	}
	return []string{"shutdown", "-h", "now"}
}

func getRebootCommand() []string {

	if runtime.GOOS == "windows" {
		return []string{"shutdown.exe", "/r", "/t", "0"}
	}
	return []string{"shutdown", "-r", "now"}
}

func Shutdown() {
	cmd := getShutdownCommand()
	err := exec.Command(cmd[0], cmd[1:]...).Run()
	if err != nil {
		log.Err(err).Msg("Failed to shutdown")
		return
	}
}

func Reboot() {
	cmd := getRebootCommand()
	err := exec.Command(cmd[0], cmd[1:]...).Run()
	if err != nil {
		log.Err(err).Msg("Failed to reboot")
		return
	}
}
