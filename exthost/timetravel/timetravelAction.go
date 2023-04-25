/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package timetravel

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-host/exthost"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"runtime"
	"time"
)

type timeTravelAction struct{}

type ActionState struct {
	DisableNtp    bool
	Offset        time.Duration
	OffsetApplied bool
}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[ActionState]         = (*timeTravelAction)(nil)
	_ action_kit_sdk.ActionWithStop[ActionState] = (*timeTravelAction)(nil) // Optional, needed when the action needs a stop method
)

func NewTimetravelAction() action_kit_sdk.Action[ActionState] {
	return &timeTravelAction{}
}

func (l *timeTravelAction) NewEmptyState() ActionState {
	return ActionState{}
}

// Describe returns the action description for the platform with all required information.
func (l *timeTravelAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          timeTravelActionID,
		Label:       "Time Travel",
		Description: "Change the system time by the given offset.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(timeTravelIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			// The target type this action is for
			TargetType: exthost.TargetID,
			// You can provide a list of target templates to help the user select targets.
			// A template can be used to pre-fill a selection
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label: "by host name",
					Query: "host.hostname=\"\"",
				},
			}),
		}),
		// Category for the targets to appear in
		Category: extutil.Ptr("State"),

		// To clarify the purpose of the action, you can set a kind.
		//   Attack: Will cause harm to targets
		//   Check: Will perform checks on the targets
		//   LoadTest: Will perform load tests on the targets
		//   Other
		Kind: action_kit_api.Attack,

		// How the action is controlled over time.
		//   External: The agent takes care and calls stop then the time has passed. Requires a duration parameter. Use this when the duration is known in advance.
		//   Internal: The action has to implement the status endpoint to signal when the action is done. Use this when the duration is not known in advance.
		//   Instantaneous: The action is done immediately. Use this for actions that happen immediately, e.g. a reboot.
		TimeControl: action_kit_api.External,

		// The parameters for the action
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "offset",
				Label:        "Offset",
				Description:  extutil.Ptr("The offset to the current time."),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("60m"),
				Required:     extutil.Ptr(true),
				Order:        extutil.Ptr(1),
			},
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr("How long should CPU be stressed?"),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
				Order:        extutil.Ptr(2),
			}, {
				Name:         "disableNtp",
				Label:        "Disable NTP",
				Description:  extutil.Ptr("Prevent NTP from correcting time during attack."),
				Type:         action_kit_api.Boolean,
				DefaultValue: extutil.Ptr("true"),
				Required:     extutil.Ptr(false),
				Advanced:     extutil.Ptr(true),
				Order:        extutil.Ptr(1),
			},
		},
		Stop: extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
	}
}

// Prepare is called before the action is started.
// It can be used to validate the parameters and prepare the action.
// It must not cause any harmful effects.
// The passed in state is included in the subsequent calls to start/status/stop.
// So the state should contain all information needed to execute the action and even more important: to be able to stop it.
func (l *timeTravelAction) Prepare(_ context.Context, state *ActionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	parsedOffset := exthost.ToUInt64(request.Config["offset"])
	if parsedOffset == 0 {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Offset is required",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, nil
	}
	offsetInSec := parsedOffset / 1000
	if offsetInSec != 0 {
		offset := time.Duration(offsetInSec * 1000 * 1000 * 1000)
		state.Offset = offset
	} else {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Offset is required",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, nil
	}
	disableNtp := exthost.ToBool(request.Config["disableNtp"])
	state.DisableNtp = disableNtp
	if !isUnixLike() {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Cannot run on non-unix-like systems",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, nil
	}

	return nil, nil
}

func isUnixLike() bool {
	return runtime.GOOS != "windows"
}

// Start is called to start the action
// You can mutate the state here.
// You can use the result to return messages/errors/metrics or artifacts
func (l *timeTravelAction) Start(_ context.Context, state *ActionState) (*action_kit_api.StartResult, error) {
	if state.DisableNtp {
		log.Info().Msg("Disabling NTP")
		err := adjustNtpTrafficRules(false)
		if err != nil {
			log.Error().Err(err).Msg("Failed to disable NTP")
			return extutil.Ptr(action_kit_api.StartResult{
				Error: extutil.Ptr(action_kit_api.ActionKitError{
					Title:  "Failed to disable NTP",
					Status: extutil.Ptr(action_kit_api.Errored),
				}),
			}), nil
		}
	}
	log.Info().Msg("Adjusting time")
	err := AdjustTime(state.Offset, false)
	if err != nil {
		log.Error().Err(err).Msg("Failed to adjust time")
		return extutil.Ptr(action_kit_api.StartResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Failed to adjust time",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}), nil
	}
	state.OffsetApplied = true
	return nil, nil
}

// Stop is called to stop the action
// It will be called even if the start method did not complete successfully.
// It should be implemented in a immutable way, as the agent might to retries if the stop method timeouts.
// You can use the result to return messages/errors/metrics or artifacts
func (l *timeTravelAction) Stop(_ context.Context, state *ActionState) (*action_kit_api.StopResult, error) {
	log.Info().Msg("Stopping action")
	log.Debug().Msgf("Offset was applied: %v", state.OffsetApplied)
	if state.OffsetApplied {
		log.Info().Msg("Adjusting time back")
		err := AdjustTime(state.Offset, true)
		if err != nil {
			log.Error().Err(err).Msg("Failed to adjust time")
			return nil, err
		}
		if state.DisableNtp {
			log.Info().Msg("Enabling NTP")
			err := adjustNtpTrafficRules(true)
			if err != nil {
				log.Error().Err(err).Msg("Failed to enable NTP")
				return nil, err
			}
		}
		state.OffsetApplied = false
	}
	return nil, nil
}
