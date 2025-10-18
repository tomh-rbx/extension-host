// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Roblox Corporation
// Author: Tom Handal <thandal@roblox.com>

package exthost

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_commons/ociruntime"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-host/config"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"golang.org/x/sync/syncmap"
)

const (
	nicFlapActionID = BaseActionID + ".network_nic_flap"
	nicFlapIcon     = "data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20fill-rule%3D%22evenodd%22%20clip-rule%3D%22evenodd%22%20d%3D%22M12%202C6.48%202%202%206.48%202%2012s4.48%2010%2010%2010%2010-4.48%2010-10S17.52%202%2012%202zM4%2012c0-4.42%203.58-8%208-8s8%203.58%208%208-3.58%208-8%208-8-3.58-8-8zm8-6c-3.31%200-6%202.69-6%206s2.69%206%206%206%206-2.69%206-6-2.69-6-6-6zm0%2010c-2.21%200-4-1.79-4-4s1.79-4%204-4%204%201.79%204%204-1.79%204-4%204z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A"
)

type nicFlapAction struct {
	ociRuntime ociruntime.OciRuntime
	flappers   syncmap.Map // Map of execution ID to *NicFlapActionState for tracking active flappers
}

type NicFlapActionState struct {
	ExecutionId   uuid.UUID
	Duration      time.Duration
	Frequency     time.Duration
	FlapDuration  time.Duration
	Jitter        bool
	Interface     string
	ExtensionPort int
	HealthPort    int
	IsFlapping    bool
	LastFlapTime  time.Time
	NextFlapTime  time.Time
	StartTime     time.Time
}

// Make sure nicFlapAction implements all required interfaces
var _ action_kit_sdk.Action[NicFlapActionState] = (*nicFlapAction)(nil)
var _ action_kit_sdk.ActionWithStop[NicFlapActionState] = (*nicFlapAction)(nil)
var _ action_kit_sdk.ActionWithStatus[NicFlapActionState] = (*nicFlapAction)(nil)

func NewNetworkNicFlapAction(r ociruntime.OciRuntime) action_kit_sdk.Action[NicFlapActionState] {
	return &nicFlapAction{
		ociRuntime: r,
	}
}

func (a *nicFlapAction) NewEmptyState() NicFlapActionState {
	return NicFlapActionState{}
}

func (a *nicFlapAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          nicFlapActionID,
		Label:       "NIC Flapping",
		Description: "Simulates network interface flapping by periodically dropping all packets",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(nicFlapIcon),
		TargetSelection: &action_kit_api.TargetSelection{
			TargetType:         targetID,
			SelectionTemplates: &targetSelectionTemplates,
		},
		Technology:  extutil.Ptr("Linux Host"),
		Category:    extutil.Ptr("Network"),
		Kind:        action_kit_api.Attack,
		TimeControl: action_kit_api.TimeControlExternal,
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr("How long should the NIC flapping experiment run?"),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("60s"),
				Required:     extutil.Ptr(true),
				Order:        extutil.Ptr(1),
			},
			{
				Name:         "frequency",
				Label:        "Frequency",
				Description:  extutil.Ptr("How often the NIC will flap (time between flap cycles)."),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("15s"),
				Required:     extutil.Ptr(true),
				Order:        extutil.Ptr(2),
			},
			{
				Name:         "flap_duration",
				Label:        "Flap Duration",
				Description:  extutil.Ptr("How long the NIC will be down for each flap."),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("5s"),
				Required:     extutil.Ptr(true),
				Order:        extutil.Ptr(3),
			},
			{
				Name:         "jitter",
				Label:        "Jitter",
				Description:  extutil.Ptr("Add +/- 30% random variation to frequency and flap duration."),
				Type:         action_kit_api.ActionParameterTypeBoolean,
				DefaultValue: extutil.Ptr("false"),
				Required:     extutil.Ptr(false),
				Order:        extutil.Ptr(4),
			},
			{
				Name:         "interface",
				Label:        "Network Interface",
				Description:  extutil.Ptr("Network interface to flap. If not specified, the first available interface will be used."),
				Type:         action_kit_api.ActionParameterTypeString,
				DefaultValue: extutil.Ptr(""),
				Required:     extutil.Ptr(false),
				Order:        extutil.Ptr(5),
			},
		},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("2s"), // Poll every 2 seconds to catch flapping events
		}),
		Widgets: extutil.Ptr([]action_kit_api.Widget{
			action_kit_api.StateOverTimeWidget{
				Type:  action_kit_api.ComSteadybitWidgetStateOverTime,
				Title: "NIC Status",
				Identity: action_kit_api.StateOverTimeWidgetIdentityConfig{
					From: "nic.interface",
				},
				Label: action_kit_api.StateOverTimeWidgetLabelConfig{
					From: "nic.interface",
				},
				State: action_kit_api.StateOverTimeWidgetStateConfig{
					From: "nic.state",
				},
				Tooltip: action_kit_api.StateOverTimeWidgetTooltipConfig{
					From: "nic.tooltip",
				},
				Value: extutil.Ptr(action_kit_api.StateOverTimeWidgetValueConfig{
					Hide: extutil.Ptr(false),
				}),
			},
		}),
	}
}

func (a *nicFlapAction) Prepare(ctx context.Context, state *NicFlapActionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	_, err := CheckTargetHostname(request.Target.Attributes)
	if err != nil {
		return nil, err
	}

	// Parse parameters
	duration := time.Duration(extutil.ToInt64(request.Config["duration"])) * time.Millisecond
	frequency := time.Duration(extutil.ToInt64(request.Config["frequency"])) * time.Millisecond
	flapDuration := time.Duration(extutil.ToInt64(request.Config["flap_duration"])) * time.Millisecond

	jitter := extutil.ToBool(request.Config["jitter"])

	// Validate frequency vs flap duration, accounting for jitter
	if jitter {
		// With jitter: ensure min frequency >= 2.6 * max flap duration
		// This prevents overlapping flaps even in worst-case jitter scenarios
		minFrequency := float64(frequency) * 0.7       // -30% jitter
		maxFlapDuration := float64(flapDuration) * 1.3 // +30% jitter
		if minFrequency < maxFlapDuration*2 {
			return nil, extension_kit.ToError("With jitter enabled, frequency must be at least 2.6x the flap duration to prevent overlapping flaps", nil)
		}
	} else {
		// Without jitter: flap duration must be no more than half of frequency
		if flapDuration > frequency/2 {
			return nil, extension_kit.ToError("Flap duration must be no more than half of the frequency", nil)
		}
	}

	// Get network interface from target attributes
	interfaces := request.Target.Attributes["host.nic"]
	if len(interfaces) == 0 {
		return nil, extension_kit.ToError("No network interfaces found on target host", nil)
	}

	// Get interface selection from config or use best available
	interfaceName := extutil.ToString(request.Config["interface"])
	if interfaceName == "" {
		// Prefer physical interfaces over virtual ones
		interfaceName = a.selectBestInterface(interfaces)
	} else {
		// Validate that the specified interface exists in the discovered interfaces
		interfaceExists := false
		for _, iface := range interfaces {
			if iface == interfaceName {
				interfaceExists = true
				break
			}
		}
		if !interfaceExists {
			return nil, extension_kit.ToError(fmt.Sprintf("Interface %s not found on target host. Available interfaces: %v", interfaceName, interfaces), nil)
		}
	}

	// Check if interface exists on the system
	if err := a.checkInterfaceExists(interfaceName); err != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Interface %s does not exist on the system", interfaceName), err)
	}

	state.ExecutionId = request.ExecutionId
	state.Duration = duration
	state.Frequency = frequency
	state.FlapDuration = flapDuration
	state.Jitter = jitter
	state.Interface = interfaceName
	state.ExtensionPort = int(config.Config.Port)
	state.HealthPort = int(config.Config.HealthPort)
	state.IsFlapping = false
	state.StartTime = time.Now()

	var messages []action_kit_api.Message
	messages = append(messages, action_kit_api.Message{
		Level:   extutil.Ptr(action_kit_api.Info),
		Message: fmt.Sprintf("Will flap network interface: %s", interfaceName),
	})

	if len(interfaces) > 1 {
		messages = append(messages, action_kit_api.Message{
			Level:   extutil.Ptr(action_kit_api.Info),
			Message: fmt.Sprintf("Available interfaces on target host: %v", interfaces),
		})
	}

	return &action_kit_api.PrepareResult{
		Messages: &messages,
	}, nil
}

func (a *nicFlapAction) Start(ctx context.Context, state *NicFlapActionState) (*action_kit_api.StartResult, error) {
	// Start the flapping goroutine with a background context
	// Don't use the ctx from Start() as it gets cancelled immediately
	fmt.Printf("Starting NIC flapping goroutine for interface %s\n", state.Interface)
	go a.runNicFlapping(context.Background(), state)

	return &action_kit_api.StartResult{
		Messages: &[]action_kit_api.Message{
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Message: fmt.Sprintf("Started NIC flapping on interface %s with frequency %v and flap duration %v", state.Interface, state.Frequency, state.FlapDuration),
			},
		},
	}, nil
}

func (a *nicFlapAction) Stop(ctx context.Context, state *NicFlapActionState) (*action_kit_api.StopResult, error) {
	// Remove from active flappers first to signal the goroutine to stop
	a.flappers.Delete(state.ExecutionId)

	// Clean up NFTABLES rules immediately in case we're currently in a flap
	if err := a.cleanupNftables(state.Interface); err != nil {
		return nil, extension_kit.ToError("Failed to cleanup NFTABLES rules", err)
	}

	// Set flapping state to false to ensure status shows correct state
	state.IsFlapping = false

	return &action_kit_api.StopResult{
		Messages: &[]action_kit_api.Message{
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Message: fmt.Sprintf("Stopped NIC flapping on interface %s", state.Interface),
			},
		},
	}, nil
}

func (a *nicFlapAction) Status(ctx context.Context, state *NicFlapActionState) (*action_kit_api.StatusResult, error) {
	// Get the shared state from active flappers
	sharedStateInterface, exists := a.flappers.Load(state.ExecutionId)
	if !exists {
		fmt.Printf("Status: Flapping not active for execution %s\n", state.ExecutionId)
		return &action_kit_api.StatusResult{Completed: true}, nil
	}

	// Cast to the correct type
	sharedState, ok := sharedStateInterface.(*NicFlapActionState)
	if !ok {
		fmt.Printf("Status: Invalid state type for execution %s\n", state.ExecutionId)
		return &action_kit_api.StatusResult{Completed: true}, nil
	}

	fmt.Printf("Status: Flapping active for execution %s, IsFlapping=%v (shared), NextFlapTime=%v\n", state.ExecutionId, sharedState.IsFlapping, sharedState.NextFlapTime)

	// Check if experiment duration has been reached
	if time.Since(sharedState.StartTime) >= sharedState.Duration {
		// Remove from active flappers and mark as completed
		a.flappers.Delete(state.ExecutionId)
		return &action_kit_api.StatusResult{
			Completed: true,
			Messages: &[]action_kit_api.Message{
				{
					Level:   extutil.Ptr(action_kit_api.Info),
					Message: fmt.Sprintf("NIC flapping experiment completed after %v", sharedState.Duration),
				},
			},
		}, nil
	}

	// Return status with metrics for the widget
	now := time.Now()
	var stateValue string
	var labelValue string

	if sharedState.IsFlapping {
		stateValue = "danger" // down = danger (red)
		labelValue = "DOWN"
	} else {
		stateValue = "success" // up = success (green)
		labelValue = "UP"
	}

	// Create a single metric with all widget fields
	metrics := []action_kit_api.Metric{
		{
			Name:      extutil.Ptr("nic.state"),
			Value:     1, // Always 1 for presence
			Timestamp: now,
			Metric: map[string]string{
				"nic.interface": sharedState.Interface,
				"nic.state":     stateValue,
				"nic.label":     labelValue,
				"nic.tooltip":   fmt.Sprintf("NIC %s is %s", sharedState.Interface, labelValue),
			},
		},
	}

	return &action_kit_api.StatusResult{
		Completed: false,
		Metrics:   &metrics,
	}, nil
}

func (a *nicFlapAction) runNicFlapping(ctx context.Context, state *NicFlapActionState) {
	// Store pointer to state in active flappers for shared access
	a.flappers.Store(state.ExecutionId, state)
	fmt.Printf("Goroutine: Stored state pointer for execution %s, IsFlapping=%v\n", state.ExecutionId, state.IsFlapping)

	// Start first flap immediately, then calculate subsequent flap times
	state.NextFlapTime = time.Now()
	fmt.Printf("NIC flapping goroutine started for interface %s, first flap starting immediately\n", state.Interface)

	// Create a timer for the total experiment duration
	experimentTimer := time.NewTimer(state.Duration)
	defer experimentTimer.Stop()

	// Create a ticker to check for stop signals periodically
	stopCheckTicker := time.NewTicker(1 * time.Second)
	defer stopCheckTicker.Stop()

	for {
		select {
		case <-experimentTimer.C:
			// Experiment duration reached, clean up and return
			fmt.Printf("NIC flapping goroutine stopped due to experiment duration reached\n")
			a.cleanupNftables(state.Interface)
			return
		case <-stopCheckTicker.C:
			// Check if we've been stopped (removed from active flappers)
			if _, exists := a.flappers.Load(state.ExecutionId); !exists {
				// We've been stopped, clean up and return
				fmt.Printf("NIC flapping goroutine stopped due to removal from active flappers\n")
				a.cleanupNftables(state.Interface)
				return
			}
		case <-time.After(time.Until(state.NextFlapTime)):
			// Check if we're still within the experiment duration
			if time.Since(state.StartTime) >= state.Duration {
				// Experiment duration reached, clean up and return
				fmt.Printf("NIC flapping goroutine stopped due to experiment duration check\n")
				a.cleanupNftables(state.Interface)
				return
			}

			// Start flapping
			state.IsFlapping = true
			state.LastFlapTime = time.Now()
			fmt.Printf("Starting NIC flap on interface %s, IsFlapping set to %v\n", state.Interface, state.IsFlapping)

			// Apply NFTABLES rules to drop packets
			if err := a.applyNftablesRules(state); err != nil {
				// Log error but continue
				fmt.Printf("Failed to apply NFTABLES rules: %v\n", err)
				continue
			}

			// Wait for flap duration with periodic checks for stop signal
			flapDuration := a.calculateWithJitter(state.FlapDuration, state.Jitter)
			flapTimer := time.NewTimer(flapDuration)
			defer flapTimer.Stop()

			select {
			case <-ctx.Done():
				// Stop was called during flap, clean up immediately
				state.IsFlapping = false
				a.cleanupNftables(state.Interface)
				return
			case <-flapTimer.C:
				// Flap duration completed normally
			}

			// Stop flapping
			state.IsFlapping = false
			fmt.Printf("Stopped NIC flap on interface %s, IsFlapping set to %v\n", state.Interface, state.IsFlapping)

			// Remove NFTABLES rules
			if err := a.cleanupNftables(state.Interface); err != nil {
				// Log error but continue
				fmt.Printf("Failed to cleanup NFTABLES rules: %v\n", err)
				continue
			}

			// Calculate next flap time
			state.NextFlapTime = time.Now().Add(a.calculateWithJitter(state.Frequency, state.Jitter))
		}
	}
}

func (a *nicFlapAction) calculateWithJitter(duration time.Duration, jitter bool) time.Duration {
	if !jitter {
		return duration
	}

	// Add +/- 30% jitter
	jitterRange := float64(duration) * 0.3
	jitterValue := (rand.Float64() - 0.5) * 2 * jitterRange
	return duration + time.Duration(jitterValue)
}

func (a *nicFlapAction) checkInterfaceExists(interfaceName string) error {
	cmd := exec.Command("ip", "link", "show", interfaceName)
	return cmd.Run()
}

func (a *nicFlapAction) selectBestInterface(interfaces []string) string {
	// First, try to find the default gateway interface
	defaultGatewayInterface := a.findDefaultGatewayInterface()
	if defaultGatewayInterface != "" {
		// Verify it's in our available interfaces
		for _, iface := range interfaces {
			if iface == defaultGatewayInterface {
				return iface
			}
		}
	}

	// Fallback to physical interfaces over virtual ones
	// Priority order: eno*, ens*, eth*, then others
	physicalInterfaces := []string{}
	virtualInterfaces := []string{}

	for _, iface := range interfaces {
		if iface == "lo" {
			continue // Skip loopback
		}

		if strings.HasPrefix(iface, "eno") || strings.HasPrefix(iface, "ens") || strings.HasPrefix(iface, "eth") {
			physicalInterfaces = append(physicalInterfaces, iface)
		} else {
			virtualInterfaces = append(virtualInterfaces, iface)
		}
	}

	// Return first physical interface, or first virtual if no physical available
	if len(physicalInterfaces) > 0 {
		return physicalInterfaces[0]
	}
	if len(virtualInterfaces) > 0 {
		return virtualInterfaces[0]
	}

	// Fallback to first available (excluding lo)
	for _, iface := range interfaces {
		if iface != "lo" {
			return iface
		}
	}

	// Last resort
	return interfaces[0]
}

func (a *nicFlapAction) findDefaultGatewayInterface() string {
	// Get the default route interface from ip route
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for "default via" and extract the interface
		parts := strings.Fields(line)
		if len(parts) >= 5 && parts[0] == "default" && parts[1] == "via" {
			// Find the "dev" keyword and get the interface name
			for i, part := range parts {
				if part == "dev" && i+1 < len(parts) {
					return parts[i+1]
				}
			}
		}
	}

	return ""
}

func (a *nicFlapAction) applyNftablesRules(state *NicFlapActionState) error {
	// Create NFTABLES rules to drop all packets except those to the extension port
	tableName := "nicflap_" + state.Interface
	chainInputName := "input_" + state.Interface
	chainOutputName := "output_" + state.Interface

	// Create table
	cmd := exec.Command("nft", "add", "table", "inet", tableName)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Create input chain
	cmd = exec.Command("nft", "add", "chain", "inet", tableName, chainInputName, "{", "type", "filter", "hook", "input", "priority", "0", ";", "}")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Create output chain
	cmd = exec.Command("nft", "add", "chain", "inet", tableName, chainOutputName, "{", "type", "filter", "hook", "output", "priority", "0", ";", "}")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Add rules to allow extension and health port traffic
	extensionPortStr := strconv.Itoa(state.ExtensionPort)
	healthPortStr := strconv.Itoa(state.HealthPort)

	// Input rule: allow incoming traffic to extension port
	cmd = exec.Command("nft", "add", "rule", "inet", tableName, chainInputName, "iif", state.Interface, "tcp", "dport", extensionPortStr, "accept")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Input rule: allow incoming traffic to health port
	cmd = exec.Command("nft", "add", "rule", "inet", tableName, chainInputName, "iif", state.Interface, "tcp", "dport", healthPortStr, "accept")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Input rule: drop all other incoming traffic on this interface
	cmd = exec.Command("nft", "add", "rule", "inet", tableName, chainInputName, "iif", state.Interface, "drop")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Output rule: allow outgoing traffic from extension port
	cmd = exec.Command("nft", "add", "rule", "inet", tableName, chainOutputName, "oif", state.Interface, "tcp", "sport", extensionPortStr, "accept")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Output rule: allow outgoing traffic from health port
	cmd = exec.Command("nft", "add", "rule", "inet", tableName, chainOutputName, "oif", state.Interface, "tcp", "sport", healthPortStr, "accept")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Output rule: drop all other outgoing traffic on this interface
	cmd = exec.Command("nft", "add", "rule", "inet", tableName, chainOutputName, "oif", state.Interface, "drop")
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (a *nicFlapAction) cleanupNftables(interfaceName string) error {
	tableName := "nicflap_" + interfaceName

	// Flush and delete table - ignore errors as table might not exist
	cmd := exec.Command("nft", "flush", "table", "inet", tableName)
	cmd.Run() // Ignore errors

	cmd = exec.Command("nft", "delete", "table", "inet", tableName)
	err := cmd.Run()

	// If table doesn't exist, that's not an error
	if err != nil {
		// Check if it's because the table doesn't exist
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			// Table might not exist, which is fine
			return nil
		}
		return err
	}

	return nil
}
