//go:build !linux

package resources

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-host/exthost/common"
	"os"
	"os/exec"
)

func AdjustOOMScoreAdj() {
	log.Info().Msg("Adjusting OOM score adj")
	myPid := os.Getpid()
	path := "/proc/" + fmt.Sprintf("%d", myPid) + "/oom_score_adj"
	out, err := common.RunAsRootWithResult("echo", "-997", ">", path)
	if err != nil {
		log.Error().Err(err).Msg("Failed to adjust OOM score adj")
	}
	log.Info().Msgf("OOM score adj result: %s", out)
	output, err := exec.Command("cat", path).CombinedOutput()
	if err != nil {
		return
	}
	log.Info().Msgf("OOM score adj: %s", output)
}
