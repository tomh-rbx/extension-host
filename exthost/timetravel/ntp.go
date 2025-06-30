// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package timetravel

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_commons/network"
)

func AdjustNtpTrafficRules(ctx context.Context, runner network.CommandRunner, allowNtpTraffic bool) error {
	opts := &network.BlackholeOpts{
		IpProto: network.IpProtoUdp,
		Filter: network.Filter{
			Include: network.NewNetWithPortRanges(network.NetAny, network.PortRange{From: 123, To: 123}),
		},
	}

	if allowNtpTraffic {
		return network.Revert(ctx, runner, opts)
	} else {
		return network.Apply(ctx, runner, opts)
	}
}
