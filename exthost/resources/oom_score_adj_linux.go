//go:build linux

package resources

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_commons/utils"
	"os"
	"path/filepath"
	"strconv"
)

func AdjustOOMScoreAdj() {
	log.Debug().Msg("Adjusting OOM score")
	path := filepath.Join("/proc", strconv.Itoa(os.Getpid()), "oom_score_adj")
	if err := utils.RootCommandContext(context.Background(), "sh", "-c", "echo -997 > "+path).Run(); err != nil {
		log.Warn().Err(err).Msg("Failed to adjust OOM score")
	}

	if oomScore, err := os.ReadFile(path); err == nil {
		log.Debug().Msgf("%s: %s", path, oomScore)
	}
}
