// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
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
