package shutdown

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-host/exthost/common"
	"os"
	"os/exec"
	"runtime"
)

type Command interface {
	IsShutdownCommandExecutable() bool
	Shutdown() error
	Reboot() error
}

type CommandImpl struct{}

func NewCommand() Command {
	return &CommandImpl{}
}

func (c *CommandImpl) IsShutdownCommandExecutable() bool {
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
		if !c.isExecAny(info.Mode()) {
			log.Debug().Msgf("Shutdown is not executable")
			return false
		}
		return true
	}
}

func (c *CommandImpl) isExecAny(mode os.FileMode) bool {
	return mode&0111 != 0
}

func (c *CommandImpl) getShutdownCommand() []string {

	if runtime.GOOS == "windows" {
		return []string{"shutdown.exe", "/s", "/t", "0"}
	}
	return []string{"shutdown", "-h", "now"}
}

func (c *CommandImpl) getRebootCommand() []string {

	if runtime.GOOS == "windows" {
		return []string{"shutdown.exe", "/r", "/t", "0"}
	}
	return []string{"shutdown", "-r", "now"}
}

func (c *CommandImpl) Shutdown() error {
	cmd := c.getShutdownCommand()
	err := common.RunAsRoot(cmd[0], cmd[1:]...)
	if err != nil {
		log.Err(err).Msg("Failed to shutdown")
		return err
	}
	return nil
}

func (c *CommandImpl) Reboot() error {
	cmd := c.getRebootCommand()
	err := common.RunAsRoot(cmd[0], cmd[1:]...)
	if err != nil {
		log.Err(err).Msg("Failed to reboot")
		return err
	}
	return nil
}
