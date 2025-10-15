package exthost

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_commons/network"
	"github.com/steadybit/action-kit/go/action_kit_commons/ociruntime"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

func NewNetworkDNSErrorInjectionAction(r ociruntime.OciRuntime) action_kit_sdk.Action[NetworkActionState] {
	return &networkAction{
		ociRuntime:   r,
		optsProvider: dnsErrorInjection(r),
		optsDecoder:  dnsErrorInjectionDecode,
		description:  getNetworkDNSErrorInjectionDescription(),
	}
}

func getNetworkDNSErrorInjectionDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          fmt.Sprintf("%s.network_dns_error_injection", BaseActionID),
		Label:       "DNS Error Injection",
		Description: "Inject DNS errors (NXDOMAIN/SERVFAIL) into DNS queries using eBPF.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(delayIcon),
		TargetSelection: &action_kit_api.TargetSelection{
			TargetType:         targetID,
			SelectionTemplates: &targetSelectionTemplates,
		},
		Technology:  extutil.Ptr("Linux Host"),
		Category:    extutil.Ptr("Network"),
		Kind:        action_kit_api.Attack,
		TimeControl: action_kit_api.TimeControlExternal,
		Parameters: append(
			commonNetworkParameters,
			action_kit_api.ActionParameter{
				Name:         "dnsErrorTypes",
				Label:        "DNS Error Types",
				Description:  extutil.Ptr("Which DNS errors to inject?"),
				Type:         action_kit_api.ActionParameterTypeStringArray,
				DefaultValue: extutil.Ptr("[\"NXDOMAIN\"]"),
				Required:     extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "NXDOMAIN",
						Value: "NXDOMAIN",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SERVFAIL",
						Value: "SERVFAIL",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Both (Random)",
						Value: "BOTH",
					},
				}),
				Order: extutil.Ptr(1),
			},
			action_kit_api.ActionParameter{
				Name:        "networkInterface",
				Label:       "Network Interface",
				Description: extutil.Ptr("Target Network Interface which should be affected. All if none specified."),
				Type:        action_kit_api.ActionParameterTypeStringArray,
				Required:    extutil.Ptr(false),
				Order:       extutil.Ptr(104),
			},
		),
	}
}

func dnsErrorInjection(r ociruntime.OciRuntime) networkOptsProvider {
	return func(ctx context.Context, sidecar network.SidecarOpts, request action_kit_api.PrepareActionRequestBody) (network.Opts, action_kit_api.Messages, error) {
		_, err := CheckTargetHostname(request.Target.Attributes)
		if err != nil {
			return nil, nil, err
		}

		errorTypes := extutil.ToStringArray(request.Config["dnsErrorTypes"])

		if len(errorTypes) == 0 {
			return nil, []action_kit_api.Message{{
				Level:   extutil.Ptr(action_kit_api.Error),
				Message: "Please select at least one DNS error type to inject.",
			}}, fmt.Errorf("no DNS error types configured")
		}

		// Validate error types
		validTypes := map[string]bool{
			"NXDOMAIN": true,
			"SERVFAIL": true,
			"BOTH":     true,
		}

		for _, errorType := range errorTypes {
			if !validTypes[errorType] {
				return nil, []action_kit_api.Message{{
					Level:   extutil.Ptr(action_kit_api.Error),
					Message: fmt.Sprintf("Invalid DNS error type: %s. Valid types are: NXDOMAIN, SERVFAIL, BOTH", errorType),
				}}, fmt.Errorf("invalid DNS error type: %s", errorType)
			}
		}

		filter, messages, err := mapToNetworkFilter(ctx, r, sidecar, request.Config, getRestrictedEndpoints(request))
		if err != nil {
			return nil, nil, err
		}

		interfaces := extutil.ToStringArray(request.Config["networkInterface"])
		if len(interfaces) == 0 {
			interfaces, err = network.ListNonLoopbackInterfaceNames(ctx, runner(r, sidecar))
			if err != nil {
				return nil, nil, err
			}
		}

		if len(interfaces) == 0 {
			return nil, nil, fmt.Errorf("no network interfaces specified")
		}

		opts := &network.DNSErrorInjectionOpts{
			Filter:     filter,
			Interfaces: interfaces,
			ErrorTypes: errorTypes,
		}

		// Validate that we have specific targets for safety
		if err := opts.ValidateTargeting(); err != nil {
			return nil, []action_kit_api.Message{{
				Level:   extutil.Ptr(action_kit_api.Error),
				Message: err.Error(),
			}}, err
		}

		return opts, messages, nil
	}
}

func dnsErrorInjectionDecode(data json.RawMessage) (network.Opts, error) {
	var opts network.DNSErrorInjectionOpts
	err := json.Unmarshal(data, &opts)
	return &opts, err
}
