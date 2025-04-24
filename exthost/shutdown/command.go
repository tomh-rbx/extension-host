// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package shutdown

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-host/exthost/common"
	"os"
	"os/exec"
	"time"
)

type commandShutdown struct {
	run      func(name string, arg ...string) error
	lookPath func(name string) (string, error)
}

func newCommandShutdown() Shutdown {
	return &commandShutdown{run: common.RunAsRoot, lookPath: exec.LookPath}
}

func (c *commandShutdown) IsAvailable() bool {
	if _, err := c.lookPath("shutdown"); err != nil {
		log.Debug().Err(err).Msgf("failed to find shutdown")
		return false
	}
	return true
}

func isExecAny(mode os.FileMode) bool {
	return mode&0111 != 0
}

func (c *commandShutdown) Shutdown() error {
	return c.runShutdown("-h")

}

func (c *commandShutdown) Reboot() error {
	return c.runShutdown("-r")
}

func (c *commandShutdown) Name() string {
	return "command"
}

func (c *commandShutdown) runShutdown(mode string) error {
	if err := c.run("shutdown", "-k", "--no-wall", "now"); err != nil {
		_ = c.run("shutdown", "-c")
		log.Err(err).Msg("failed 'shutdown -k --no-wall now'")
		return err
	}

	go func() {
		time.Sleep(3 * time.Second)
		if err := c.run("shutdown", mode, "now"); err != nil {
			log.Err(err).Msgf("failed 'shutdown %s now'", mode)
		}
	}()

	return nil
}
