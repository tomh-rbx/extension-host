// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package shutdown

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_commons/utils"
	"os/exec"
	"time"
)

type commandShutdown struct {
	run      func(ctx context.Context, name string, arg ...string) error
	lookPath func(name string) (string, error)
}

func newCommandShutdown() Shutdown {
	return &commandShutdown{run: func(ctx context.Context, name string, arg ...string) error {
		return utils.RootCommandContext(ctx, name, arg...).Run()
	}, lookPath: exec.LookPath}
}

func (c *commandShutdown) IsAvailable() bool {
	if _, err := c.lookPath("shutdown"); err != nil {
		log.Debug().Err(err).Msgf("failed to find shutdown")
		return false
	}
	return true
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
	if err := c.run(context.Background(), "shutdown", "-k", "--no-wall", "now"); err != nil {
		_ = c.run(context.Background(), "shutdown", "-c")
		log.Err(err).Msg("failed 'shutdown -k --no-wall now'")
		return err
	}

	go func() {
		time.Sleep(3 * time.Second)
		if err := c.run(context.Background(), "shutdown", mode, "now"); err != nil {
			log.Err(err).Msgf("failed 'shutdown %s now'", mode)
		}
	}()

	return nil
}
