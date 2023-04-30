// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net"
	"os/exec"
	"runtime"
)

type Interface struct {
	Index    uint     `json:"ifindex"`
	Name     string   `json:"ifname"`
	LinkType string   `json:"link_type"`
	Flags    []string `json:"flags"`
}

func (i *Interface) HasFlag(f string) bool {
	for _, flag := range i.Flags {
		if flag == f {
			return true
		}
	}
	return false
}

func ListInterfaces(ctx context.Context) ([]Interface, error) {

	var outb, errb bytes.Buffer
	var interfaces []Interface
	if runtime.GOOS == "darwin" {
		return listInterfacesMac(interfaces)
	}

	cmd := exec.CommandContext(ctx, "ip", "-json", "link", "show")
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("could not list interfaces: %w: %s", err, errb.String())
	}

	err = json.Unmarshal(outb.Bytes(), &interfaces)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal interfaces: %w", err)
	}

	log.Trace().Interface("interfaces", interfaces).Msg("listed network interfaces")
	return interfaces, nil
}

func listInterfacesMac(interfaces []Interface) ([]Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get interfaces")
		return nil, fmt.Errorf("could not list interfaces: %w", err)
	}
	for _, iface := range ifaces {
		var flags []string
		if iface.Flags&net.FlagUp != 0 {
			flags = append(flags, "UP")
		}
		if iface.Flags&net.FlagBroadcast != 0 {
			flags = append(flags, "BROADCAST")
		}
		if iface.Flags&net.FlagLoopback != 0 {
			flags = append(flags, "LOOPBACK")
		}
		if iface.Flags&net.FlagPointToPoint != 0 {
			flags = append(flags, "POINTTOPOINT")
		}
		if iface.Flags&net.FlagMulticast != 0 {
			flags = append(flags, "MULTICAST")
		}
		interfaces = append(interfaces, Interface{
			Index:    uint(iface.Index),
			Name:     iface.Name,
			LinkType: iface.HardwareAddr.String(),
			Flags:    flags,
		})
	}
	return interfaces, nil
}
