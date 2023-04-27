package resources

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
)

type StressActionState struct {
	StressNGArgs []string
	Pid          int
}

func Start(state *StressActionState) (*action_kit_api.StartResult, error) {
	pid, err := StartStressNG(state.StressNGArgs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start stress-ng")
		return nil, err
	}
	log.Info().Int("Pid", pid).Msg("Started stress-ng")
	state.Pid = pid
	return nil, nil
}

func Stop(state *StressActionState) (*action_kit_api.StopResult, error) {
	if state.Pid != 0 {
		log.Info().Int("Pid", state.Pid).Msg("Stopping stress-ng")
		err := StopStressNG(state.Pid)
		if err != nil {
			log.Error().Err(err).Int("Pid", state.Pid).Msg("Failed to stop stress-ng")
			return nil, err
		}
		state.Pid = 0
	}
	return nil, nil
}
