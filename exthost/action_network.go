// Copyright 2025 steadybit GmbH. All rights reserved.

package exthost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_commons/network"
	"github.com/steadybit/action-kit/go/action_kit_commons/ociruntime"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-host/config"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extutil"
)

type networkOptsProvider func(ctx context.Context, sidecar network.SidecarOpts, request action_kit_api.PrepareActionRequestBody) (network.Opts, action_kit_api.Messages, error)

type networkOptsDecoder func(data json.RawMessage) (network.Opts, error)

type networkAction struct {
	ociRuntime   ociruntime.OciRuntime
	description  action_kit_api.ActionDescription
	optsProvider networkOptsProvider
	optsDecoder  networkOptsDecoder
}

type NetworkActionState struct {
	ExecutionId uuid.UUID
	NetworkOpts json.RawMessage
	Sidecar     network.SidecarOpts
}

// Make sure networkAction implements all required interfaces
var _ action_kit_sdk.Action[NetworkActionState] = (*networkAction)(nil)
var _ action_kit_sdk.ActionWithStop[NetworkActionState] = (*networkAction)(nil)

var commonNetworkParameters = []action_kit_api.ActionParameter{
	{
		Name:         "duration",
		Label:        "Duration",
		Description:  extutil.Ptr("How long should the network be affected?"),
		Type:         action_kit_api.ActionParameterTypeDuration,
		DefaultValue: extutil.Ptr("30s"),
		Required:     extutil.Ptr(true),
		Order:        extutil.Ptr(0),
	},
	{
		Name:         "hostname",
		Label:        "Hostname",
		Description:  extutil.Ptr("Restrict to/from which hosts the traffic is affected."),
		Type:         action_kit_api.ActionParameterTypeStringArray,
		DefaultValue: extutil.Ptr(""),
		Advanced:     extutil.Ptr(true),
		Order:        extutil.Ptr(101),
	},
	{
		Name:         "ip",
		Label:        "IP Address/CIDR",
		Description:  extutil.Ptr("Restrict to/from which IP addresses or blocks the traffic is affected."),
		Type:         action_kit_api.ActionParameterTypeStringArray,
		DefaultValue: extutil.Ptr(""),
		Advanced:     extutil.Ptr(true),
		Order:        extutil.Ptr(102),
	},
	{
		Name:         "port",
		Label:        "Ports",
		Description:  extutil.Ptr("Restrict to/from which ports the traffic is affected."),
		Type:         action_kit_api.ActionParameterTypeStringArray,
		DefaultValue: extutil.Ptr(""),
		Advanced:     extutil.Ptr(true),
		Order:        extutil.Ptr(103),
	},
}

func (a *networkAction) NewEmptyState() NetworkActionState {
	return NetworkActionState{}
}

func (a *networkAction) Describe() action_kit_api.ActionDescription {
	return a.description
}

func (a *networkAction) Prepare(ctx context.Context, state *NetworkActionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	_, err := CheckTargetHostname(request.Target.Attributes)
	if err != nil {
		return nil, err
	}

	initProcess, err := ociruntime.ReadLinuxProcessInfo(ctx, 1, specs.NetworkNamespace)
	if err != nil {
		return nil, extension_kit.ToError("Failed to read root process infos.", err)
	}

	state.Sidecar = network.SidecarOpts{
		TargetProcess: initProcess,
		IdSuffix:      "host",
		ExecutionId:   request.ExecutionId,
	}
	state.ExecutionId = request.ExecutionId

	opts, messages, err := a.optsProvider(ctx, state.Sidecar, request)
	if err != nil {
		return nil, extension_kit.WrapError(err)
	}

	rawOpts, err := json.Marshal(opts)
	if err != nil {
		return nil, extension_kit.ToError("Failed to serialize network settings.", err)
	}

	state.NetworkOpts = rawOpts

	return &action_kit_api.PrepareResult{Messages: &messages}, nil
}

func (a *networkAction) Start(ctx context.Context, state *NetworkActionState) (*action_kit_api.StartResult, error) {
	opts, err := a.optsDecoder(state.NetworkOpts)
	if err != nil {
		return nil, extension_kit.ToError("Failed to deserialize network settings.", err)
	}

	result := action_kit_api.StartResult{Messages: &action_kit_api.Messages{
		{
			Level:   extutil.Ptr(action_kit_api.Info),
			Message: opts.String(),
		},
	}}

	err = network.Apply(ctx, runner(a.ociRuntime, state.Sidecar), opts)
	if err != nil {
		var toomany *network.ErrTooManyTcCommands
		if errors.As(err, &toomany) {
			result.Messages = extutil.Ptr(append(*result.Messages, action_kit_api.Message{
				Level:   extutil.Ptr(action_kit_api.Error),
				Message: fmt.Sprintf("Too many tc commands (%d) generated. This happens when too many excludes for steadybit agent and extensions are needed. Please configure a more specific attack by adding ports, and/or CIDRs to the parameters.", toomany.Count),
			}))
			return &result, nil
		}
		return &result, extension_kit.ToError("Failed to apply network settings.", err)
	}

	return &result, nil
}

func (a *networkAction) Stop(ctx context.Context, state *NetworkActionState) (*action_kit_api.StopResult, error) {
	opts, err := a.optsDecoder(state.NetworkOpts)
	if err != nil {
		return nil, extension_kit.ToError("Failed to deserialize network settings.", err)
	}

	if err := network.Revert(ctx, runner(a.ociRuntime, state.Sidecar), opts); err != nil {
		return nil, extension_kit.ToError("Failed to revert network settings.", err)
	}

	return nil, nil
}

func runner(r ociruntime.OciRuntime, sidecar network.SidecarOpts) network.CommandRunner {
	if config.Config.DisableRunc {
		return network.NewProcessRunner()
	}
	return network.NewRuncRunner(r, sidecar)
}

func parsePortRanges(raw []string) ([]network.PortRange, error) {
	if raw == nil {
		return nil, nil
	}

	var ranges []network.PortRange

	for _, r := range raw {
		if len(r) == 0 {
			continue
		}
		parsed, err := network.ParsePortRange(r)
		if err != nil {
			return nil, err
		}
		ranges = append(ranges, parsed)
	}

	return ranges, nil
}

func hostnameResolver(r ociruntime.OciRuntime, sidecar network.SidecarOpts) *network.HostnameResolver {
	if config.Config.DisableRunc {
		return &network.HostnameResolver{Dig: &network.CommandDigRunner{}}
	}
	return &network.HostnameResolver{Dig: &network.RuncDigRunner{Runc: r, Sidecar: sidecar}}
}

func mapToNetworkFilter(ctx context.Context, r ociruntime.OciRuntime, sidecar network.SidecarOpts, actionConfig map[string]interface{}, restrictedEndpoints []action_kit_api.RestrictedEndpoint) (network.Filter, action_kit_api.Messages, error) {
	includeCidrs, unresolved := network.ParseCIDRs(append(
		extutil.ToStringArray(actionConfig["ip"]),
		extutil.ToStringArray(actionConfig["hostname"])...,
	))

	resolved, err := hostnameResolver(r, sidecar).Resolve(ctx, unresolved...)
	if err != nil {
		return network.Filter{}, nil, err
	}
	includeCidrs = append(includeCidrs, network.IpsToNets(resolved)...)

	//if no hostname/ip specified we affect all ips
	if len(includeCidrs) == 0 {
		includeCidrs = network.NetAny
	}

	portRanges, err := parsePortRanges(extutil.ToStringArray(actionConfig["port"]))
	if err != nil {
		return network.Filter{}, nil, err
	}
	if len(portRanges) == 0 {
		//if no hostname/ip specified we affect all ports
		portRanges = []network.PortRange{network.PortRangeAny}
	}

	includes := network.NewNetWithPortRanges(includeCidrs, portRanges...)
	for _, i := range includes {
		i.Comment = "parameters"
	}

	slices.SortFunc(includes, network.NetWithPortRange.Compare)

	excludes, err := toExcludes(restrictedEndpoints)
	if err != nil {
		return network.Filter{}, nil, err
	}

	excludes = append(excludes, network.ComputeExcludesForOwnIpAndPorts(config.Config.Port, config.Config.HealthPort)...)

	slices.SortFunc(excludes, network.NetWithPortRange.Compare)

	var messages []action_kit_api.Message
	excludes, condensed := condenseExcludes(excludes)
	if condensed {
		messages = append(messages, action_kit_api.Message{
			Level: extutil.Ptr(action_kit_api.Warn),
			Message: "Some excludes (to protect agent and extensions) were aggregated to reduce the number of tc commands necessary." +
				"This may lead to less specific exclude rules, some traffic might not be affected, as expected. " +
				"You can avoid this by configuring a more specific attack (e.g. by specifying ports or CIDRs).",
		})
	}

	return network.Filter{Include: includes, Exclude: excludes}, messages, nil
}

func condenseExcludes(excludes []network.NetWithPortRange) ([]network.NetWithPortRange, bool) {
	l := len(excludes)
	excludes = network.CondenseNetWithPortRange(excludes, 500)
	return excludes, l != len(excludes)
}

func toExcludes(restrictedEndpoints []action_kit_api.RestrictedEndpoint) ([]network.NetWithPortRange, error) {
	var excludes []network.NetWithPortRange

	for _, restrictedEndpoint := range restrictedEndpoints {
		_, cidr, err := net.ParseCIDR(restrictedEndpoint.Cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid cidr %s: %w", restrictedEndpoint.Cidr, err)
		}

		nwps := network.NewNetWithPortRanges([]net.IPNet{*cidr}, network.PortRange{From: uint16(restrictedEndpoint.PortMin), To: uint16(restrictedEndpoint.PortMax)})
		for i := range nwps {
			var sb strings.Builder
			if restrictedEndpoint.Name != "" {
				sb.WriteString(restrictedEndpoint.Name)
				sb.WriteString(" ")
			}
			if restrictedEndpoint.Url != "" {
				sb.WriteString(restrictedEndpoint.Url)
			}
			nwps[i].Comment = strings.TrimSpace(sb.String())
		}

		excludes = append(excludes, nwps...)
	}
	return excludes, nil
}
