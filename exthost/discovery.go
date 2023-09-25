/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package exthost

import (
	"github.com/elastic/go-sysinfo"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_commons/networkutils"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
  "github.com/steadybit/discovery-kit/go/discovery_kit_commons"
  "github.com/steadybit/extension-host/config"
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
		TargetEnrichmentRules: []discovery_kit_api.DescribingEndpointReference{},
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
				Attribute: "host.ipv6",
				Label: discovery_kit_api.PluralLabel{
					One:   "IPv6",
					Other: "IPv6s",
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
	hostname, _ := os.Hostname()
	target := discovery_kit_api.Target{
		Id:         hostname,
		TargetType: TargetID,
		Label:      hostname,
		Attributes: map[string][]string{
			"host.hostname": {hostname},
			"host.nic":      networkutils.GetOwnNetworkInterfaces(),
		},
	}

	var ownIpV4s, ownIpV6s []string
	for _, ip := range networkutils.GetOwnIPs() {
		if ipv4 := ip.To4(); ipv4 != nil {
			ownIpV4s = append(ownIpV4s, ipv4.String())
		} else if ipv6 := ip.To16(); ipv6 != nil {
			ownIpV6s = append(ownIpV6s, ipv6.String())
		}
	}
	if len(ownIpV4s) > 0 {
		target.Attributes["host.ipv4"] = ownIpV4s
	}
	if len(ownIpV6s) > 0 {
		target.Attributes["host.ipv6"] = ownIpV6s
	}

	if host, err := sysinfo.Host(); err == nil {
		target.Attributes["host.os.family"] = []string{host.Info().OS.Family}
		target.Attributes["host.os.manufacturer"] = []string{host.Info().OS.Name}
		target.Attributes["host.os.version"] = []string{host.Info().OS.Version}

		if fqdn, err := host.FQDN(); err == nil {
			target.Attributes["host.domainname"] = []string{fqdn}
		} else {
			target.Attributes["host.domainname"] = []string{host.Info().Hostname}
		}
	} else {
		log.Error().Err(err).Msg("Failed to get host info")
	}

	for key, value := range getEnvironmentVariables() {
		target.Attributes["host.env."+key] = []string{value}
	}
	for key, value := range getLabels() {
		target.Attributes["host.label."+key] = []string{value}
	}

  targets := []discovery_kit_api.Target{target}
  return discovery_kit_commons.ApplyAttributeExcludes(targets, config.Config.DiscoveryAttributesExcludesHost)
}
