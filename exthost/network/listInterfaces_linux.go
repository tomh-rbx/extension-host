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
)

func ListInterfaces(ctx context.Context) ([]Interface, error) {
	var outb, errb bytes.Buffer
	var interfaces []Interface

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
