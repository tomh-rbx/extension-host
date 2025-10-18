package exthost

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_commons/network"
	"github.com/steadybit/action-kit/go/action_kit_commons/ociruntime"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

// Ensure dnsErrorInjectionAction implements the required interfaces
var _ action_kit_sdk.Action[NetworkActionState] = (*dnsErrorInjectionAction)(nil)
var _ action_kit_sdk.ActionWithStatus[NetworkActionState] = (*dnsErrorInjectionAction)(nil)

type dnsErrorInjectionAction struct {
	*networkAction
}

func NewNetworkDNSErrorInjectionAction(r ociruntime.OciRuntime) action_kit_sdk.Action[NetworkActionState] {
	// Clean up any orphaned eBPF filters from previous crashes
	// This is done here rather than in main() to keep cleanup localized to the DNS error injection feature
	if err := network.CleanupOrphanedEBPFFilters(); err != nil {
		log.Warn().Err(err).Msg("Failed to cleanup orphaned eBPF filters, continuing anyway")
	}

	return &dnsErrorInjectionAction{
		networkAction: &networkAction{
			ociRuntime:   r,
			optsProvider: dnsErrorInjection(r),
			optsDecoder:  dnsErrorInjectionDecode,
			description:  getNetworkDNSErrorInjectionDescription(),
		},
	}
}

func (a *dnsErrorInjectionAction) Status(ctx context.Context, state *NetworkActionState) (*action_kit_api.StatusResult, error) {
	// Get messages from the eBPF loader
	messages, err := network.GetDNSErrorInjectionMessages(state.ExecutionId.String())
	if err != nil {
		log.Warn().Err(err).Str("execution_id", state.ExecutionId.String()).Msg("Failed to get DNS error injection messages")
		return &action_kit_api.StatusResult{
			Completed: false,
		}, nil
	}

	log.Debug().
		Str("execution_id", state.ExecutionId.String()).
		Int("message_count", len(*messages)).
		Msg("returning DNS error injection messages from Status")

	return &action_kit_api.StatusResult{
		Completed: false,
		Messages:  messages,
	}, nil
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
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("2s"),
		}),
		Widgets: extutil.Ptr([]action_kit_api.Widget{
			action_kit_api.MarkdownWidget{
				Type:        action_kit_api.ComSteadybitWidgetMarkdown,
				Title:       "DNS Error Injection Statistics",
				MessageType: "dns_stats_markdown",
				Append:      false,
			},
		}),
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr("How long should the DNS errors be injected?"),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
				Order:        extutil.Ptr(0),
			},
			{
				Name:         "dnsErrorType",
				Label:        "DNS Error Type",
				Description:  extutil.Ptr("Which DNS error to inject?"),
				Type:         action_kit_api.ActionParameterTypeString,
				DefaultValue: extutil.Ptr("NXDOMAIN"),
				Required:     extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Random",
						Value: "RANDOM",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "NXDOMAIN",
						Value: "NXDOMAIN",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "SERVFAIL",
						Value: "SERVFAIL",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "TIMEOUT",
						Value: "TIMEOUT",
					},
				}),
				Order: extutil.Ptr(1),
			},
		},
	}
}

func dnsErrorInjection(r ociruntime.OciRuntime) networkOptsProvider {
	return func(ctx context.Context, sidecar network.SidecarOpts, request action_kit_api.PrepareActionRequestBody) (network.Opts, action_kit_api.Messages, error) {
		_, err := CheckTargetHostname(request.Target.Attributes)
		if err != nil {
			return nil, nil, err
		}

		errorType := extutil.ToString(request.Config["dnsErrorType"])

		if errorType == "" {
			errorType = "NXDOMAIN" // default
		}

		// Validate error type
		validTypes := map[string]bool{
			"NXDOMAIN": true,
			"SERVFAIL": true,
			"TIMEOUT":  true,
			"RANDOM":   true,
		}

		if !validTypes[errorType] {
			return nil, []action_kit_api.Message{{
				Level:   extutil.Ptr(action_kit_api.Error),
				Message: fmt.Sprintf("Invalid DNS error type: %s. Valid types are: NXDOMAIN, SERVFAIL, TIMEOUT, RANDOM", errorType),
			}}, fmt.Errorf("invalid DNS error type: %s", errorType)
		}

		// Convert single error type to array for backwards compatibility with network.Opts
		var errorTypes []string
		if errorType == "RANDOM" {
			// Random should include all three error types, plus BOTH flag for random selection
			errorTypes = []string{"BOTH", "TIMEOUT"}
		} else {
			errorTypes = []string{errorType}
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
			Filter:      filter,
			Interfaces:  interfaces,
			ErrorTypes:  errorTypes,
			ExecutionID: request.ExecutionId.String(),
			IsContainer: false, // This is a host-level attack
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
