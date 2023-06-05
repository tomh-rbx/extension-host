/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package exthost

import (
  "github.com/elastic/go-sysinfo"
  "github.com/google/uuid"
  "github.com/rs/zerolog/log"
  "github.com/steadybit/action-kit/go/action_kit_commons/networkutils"
  "github.com/steadybit/discovery-kit/go/discovery_kit_api"
  "github.com/steadybit/extension-kit/extbuild"
  "github.com/steadybit/extension-kit/exthttp"
  "github.com/steadybit/extension-kit/extutil"
  "net/http"
  "os"
)

const discoveryBasePath = basePath + "/discovery"

func RegisterDiscoveryHandlers() {
	exthttp.RegisterHttpHandler(discoveryBasePath, exthttp.GetterAsHandler(getDiscoveryDescription))
	exthttp.RegisterHttpHandler(discoveryBasePath+"/target-description", exthttp.GetterAsHandler(getTargetDescription))
	exthttp.RegisterHttpHandler(discoveryBasePath+"/attribute-descriptions", exthttp.GetterAsHandler(getAttributeDescriptions))
	exthttp.RegisterHttpHandler(discoveryBasePath+"/discovered-targets", getDiscoveredTargets)
}

func GetDiscoveryList() discovery_kit_api.DiscoveryList {
	return discovery_kit_api.DiscoveryList{
		Discoveries: []discovery_kit_api.DescribingEndpointReference{
			{
				Method: "GET",
				Path:   discoveryBasePath,
			},
		},
		TargetTypes: []discovery_kit_api.DescribingEndpointReference{
			{
				Method: "GET",
				Path:   discoveryBasePath + "/target-description",
			},
		},
		TargetAttributes: []discovery_kit_api.DescribingEndpointReference{
			{
				Method: "GET",
				Path:   discoveryBasePath + "/attribute-descriptions",
			},
		},
	}
}

func getDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         TargetID,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			Method:       "GET",
			Path:         discoveryBasePath + "/discovered-targets",
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func getTargetDescription() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:      TargetID,
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Icon:    extutil.Ptr(targetIcon),

		// Labels used in the UI
		Label: discovery_kit_api.PluralLabel{One: "Host", Other: "Hosts"},

		// Category for the targets to appear in
		Category: extutil.Ptr("basic"),

		// Specify attributes shown in table columns and to be used for sorting
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "host.hostname"},
				{Attribute: "host.ipv4"},
				{Attribute: "aws.zone"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "host.hostname",
					Direction: "ASC",
				},
			},
		},
	}
}

func getAttributeDescriptions() discovery_kit_api.AttributeDescriptions {
	return discovery_kit_api.AttributeDescriptions{
		Attributes: []discovery_kit_api.AttributeDescription{
			{
				Attribute: "host.hostname",
				Label: discovery_kit_api.PluralLabel{
					One:   "Hostname",
					Other: "Hostnames",
				},
			}, {
				Attribute: "host.domainname",
				Label: discovery_kit_api.PluralLabel{
					One:   "Domainname",
					Other: "Domainnames",
				},
			}, {
				Attribute: "host.ipv4",
				Label: discovery_kit_api.PluralLabel{
					One:   "IPv4",
					Other: "IPv4s",
				},
			}, {
				Attribute: "host.nic",
				Label: discovery_kit_api.PluralLabel{
					One:   "NIC",
					Other: "NICs",
				},
			}, {
				Attribute: "host.os.family",
				Label: discovery_kit_api.PluralLabel{
					One:   "OS Family",
					Other: "OS Families",
				},
			}, {
				Attribute: "host.os.manufacturer",
				Label: discovery_kit_api.PluralLabel{
					One:   "OS Manufacturer",
					Other: "OS Manufacturers",
				},
			}, {
				Attribute: "host.os.version",
				Label: discovery_kit_api.PluralLabel{
					One:   "OS Version",
					Other: "OS Versions",
				},
			},
		},
	}
}

func getDiscoveredTargets(w http.ResponseWriter, _ *http.Request, _ []byte) {
	targets := getHostTarget()
	exthttp.WriteBody(w, discovery_kit_api.DiscoveredTargets{Targets: targets})
}

func getHostTarget() []discovery_kit_api.Target {
	targets := make([]discovery_kit_api.Target, 1)
	hostname, _ := os.Hostname()
	ips := networkutils.GetOwnIPs()
	id := generateID(hostname)
	nics := networkutils.GetOwnNetworkInterfaces()
	host, err := sysinfo.Host()
	var osFamily string
	var osManufacturer string
	var osVersion string

	if err != nil {
		log.Error().Err(err).Msg("Failed to get host info")
	} else {
		osFamily = host.Info().OS.Family
		osManufacturer = host.Info().OS.Name
		osVersion = host.Info().OS.Version
	}
	fqdn, err := host.FQDN()
	if err != nil {
		log.Trace().Err(err).Msg("Failed to get FQDN")
		fqdn = host.Info().Hostname
	}

	// ip adress of the host
	targets[0] = discovery_kit_api.Target{
		Id:         id,
		TargetType: TargetID,
		Label:      hostname,
		Attributes: map[string][]string{
			"host.hostname":        {hostname},
			"host.domainname":      {fqdn},
			"host.ipv4":            ips,
			"host.nic":             nics,
			"host.os.family":       {osFamily},
			"host.os.manufacturer": {osManufacturer},
			"host.os.version":      {osVersion},
		},
	}
	environmentVariables := getEnvironmentVariables()
	for key, value := range environmentVariables {
		targets[0].Attributes["host.env."+key] = []string{value}
	}
	labels := getLabels()
	for key, value := range labels {
		targets[0].Attributes["label."+key] = []string{value}
	}

	return targets
}


var id = ""
func generateID(hostname string) string {
  if id == "" {
    id = hostname + "-" +uuid.New().String()
    log.Info().Msg("Generated Target ID: " + id)
  }
  return id
}

