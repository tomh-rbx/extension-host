/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package shutdown

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-host/exthost"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

type shutdownAction struct{}

type ActionState struct {
	Reboot bool
}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[ActionState] = (*shutdownAction)(nil)
)

func NewShutdownAction() action_kit_sdk.Action[ActionState] {
	return &shutdownAction{}
}

func (l *shutdownAction) NewEmptyState() ActionState {
	return ActionState{}
}

// Describe returns the action description for the platform with all required information.
func (l *shutdownAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          shutdownActionID,
		Label:       "Shutdown Host",
		Description: "Reboots or shuts down the host.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(shutdownIcon),
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
		TimeControl: action_kit_api.Instantaneous,

		// The parameters for the action
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "reboot",
				Label:        "Reboot",
				Description:  extutil.Ptr("Should the host reboot after shutting down?"),
				Type:         action_kit_api.Boolean,
				DefaultValue: extutil.Ptr("true"),
				Required:     extutil.Ptr(true),
				Order:        extutil.Ptr(2),
			},
		},
	}
}

// Prepare is called before the action is started.
// It can be used to validate the parameters and prepare the action.
// It must not cause any harmful effects.
// The passed in state is included in the subsequent calls to start/status/stop.
// So the state should contain all information needed to execute the action and even more important: to be able to stop it.
func (l *shutdownAction) Prepare(_ context.Context, state *ActionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	reboot := extutil.ToBool(request.Config["reboot"])
	state.Reboot = reboot
	return nil, nil
}

// Start is called to start the action
// You can mutate the state here.
// You can use the result to return messages/errors/metrics or artifacts
func (l *shutdownAction) Start(_ context.Context, state *ActionState) (*action_kit_api.StartResult, error) {
	if state.Reboot {
		Reboot()
	} else {
		Shutdown()
	}
	return nil, nil
}
