package common

import (
	"context"
	"github.com/rs/zerolog/log"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func ExecuteIpTablesCommand(args ...string) error {
	log.Debug().Msg("Executing iptables command")
	log.Debug().Msg(strings.Join(args, " "))
	ctxctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctxctx, "iptables", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: 0,
			Gid: 0,
		},
	}

	cmd.Env = append(cmd.Env, "XTABLES_LOCKFILE=/tmp/xtables.lock")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(out)).Msg("Failed to execute iptables command")
		return err
	}

	return nil
}
