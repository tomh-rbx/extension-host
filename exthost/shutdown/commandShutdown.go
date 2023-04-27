package shutdown

import (
	"github.com/rs/zerolog/log"
  "os"
  "os/exec"
	"runtime"
)

func isShutdownCommandExecutable() bool {
  if runtime.GOOS == "windows" {
    _, err := exec.LookPath("shutdown.exe")
    if err != nil {
      log.Debug().Msgf("Failed to find shutdown.exe %s", err)
      return false
    }
    return true
  } else {
    path, err := exec.LookPath("shutdown")
    if err != nil {
      log.Debug().Msgf("Failed to find shutdown %s", err)
      return false
    }
    info, err := os.Stat(path)
    if err != nil {
      log.Debug().Msgf("Failed to stat shutdown %s", err)
      return false
    }
    if !IsExecAny(info.Mode()) {
      log.Debug().Msgf("Shutdown is not executable")
      return false
    }
    return true
  }
}

func IsExecAny(mode os.FileMode) bool {
  return mode&0111 != 0
}

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

func Shutdown() error{
	cmd := getShutdownCommand()
	err := exec.Command(cmd[0], cmd[1:]...).Run()
	if err != nil {
		log.Err(err).Msg("Failed to shutdown")
		return err
	}
  return nil
}

func Reboot() error{
	cmd := getRebootCommand()
	err := exec.Command(cmd[0], cmd[1:]...).Run()
	if err != nil {
		log.Err(err).Msg("Failed to reboot")
		return err
	}
  return nil
}
