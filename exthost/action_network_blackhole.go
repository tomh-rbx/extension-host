// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package exthost

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-container/pkg/networkutils"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

func NewNetworkBlackholeContainerAction() action_kit_sdk.Action[NetworkActionState] {
	return &networkAction{
		optsProvider: blackhole(),
		optsDecoder:  blackholeDecode,
		description:  getNetworkBlackholeDescription(),
	}
}

func getNetworkBlackholeDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          fmt.Sprintf("%s.network_blackhole", BaseActionID),
		Label:       "Host Block Traffic",
		Description: "Blocks network traffic (incoming and outgoing).",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(blackHoleIcon),
		TargetSelection: &action_kit_api.TargetSelection{
			TargetType:         TargetID,
			SelectionTemplates: &targetSelectionTemplates,
		},
		Category:    extutil.Ptr("network"),
		Kind:        action_kit_api.Attack,
		TimeControl: action_kit_api.External,
		Parameters:  commonNetworkParameters,
	}
}

func blackhole() networkOptsProvider {
	return func(ctx context.Context, request action_kit_api.PrepareActionRequestBody) (networkutils.Opts, error) {
		hostname := request.Target.Attributes["host.hostname"][0]

		var restrictedUrls []string
		if request.ExecutionContext != nil && request.ExecutionContext.RestrictedUrls != nil {
			restrictedUrls = *request.ExecutionContext.RestrictedUrls
		}

		filter, err := mapToNetworkFilter(ctx, hostname, request.Config, restrictedUrls)
		if err != nil {
			return nil, err
		}

		return &networkutils.BlackholeOpts{Filter: filter}, nil
	}
}

func blackholeDecode(data json.RawMessage) (networkutils.Opts, error) {
	var opts networkutils.BlackholeOpts
	err := json.Unmarshal(data, &opts)
	return &opts, err
}
