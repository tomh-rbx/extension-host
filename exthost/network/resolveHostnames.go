// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package network

import (
	"bytes"
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"net"
	"os/exec"
	"strings"
)

func ResolveHostnames(ctx context.Context, ipOrHostnames ...string) ([]string, error) {
	hostnames, ips := classifyResolved(ipOrHostnames)

	if len(hostnames) == 0 {
		return ips, nil
	}

	var sb strings.Builder
	for _, hostname := range hostnames {
		if len(hostname) == 0 {
			continue
		}
		sb.WriteString(hostname)
		sb.WriteString(" A\n")
		sb.WriteString(hostname)
		sb.WriteString(" AAAA\n")
	}
	stdin := strings.NewReader(sb.String())
	var outb, errb bytes.Buffer

	cmd := exec.CommandContext(ctx, "dig", "-f-", "+timeout=4", "+short", "+nottlid", "+noclass")
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	cmd.Stdin = stdin

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("could not resolve hostnames: %w: %s", err, errb.String())
	}

	for _, ip := range strings.Split(outb.String(), "\n") {
		ips = append(ips, strings.TrimSpace(ip))
	}

	log.Trace().Strs("ips", ips).Strs("ipOrHostnames", ipOrHostnames).Msg("resolved ips")
	return ips, nil
}

func classifyResolved(ipOrHostnames []string) (unresolved, resolved []string) {
	for _, ipOrHostnames := range ipOrHostnames {
		if ip := net.ParseIP(strings.TrimPrefix(strings.TrimSuffix(ipOrHostnames, "]"), "[")); ip == nil {
			unresolved = append(unresolved, ipOrHostnames)
		} else {
			resolved = append(resolved, ip.String())
		}
	}
	return unresolved, resolved
}
