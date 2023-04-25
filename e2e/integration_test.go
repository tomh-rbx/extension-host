// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-host/exthost"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
	"time"
)

func skipCI(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
}
func TestWithMinikube(t *testing.T) {
	skipCI(t)
	WithMinikube(t, []WithMinikubeTestCase{
		{
			Name: "target discovery",
			Test: testDiscovery,
		},
		{
			Name: "stress cpu",
			Test: testStressCpu,
		},
		{
			Name: "stress memory",
			Test: testStressMemory,
		}, {
			Name: "stress io",
			Test: testStressIo,
		},
		{
			Name: "timetravel",
			Test: testTimeTravel,
		},
	})
}

func testStressCpu(t *testing.T, m *Minikube, e *Extension) {

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
		return
	}
	target := action_kit_api.Target{
		Attributes: map[string][]string{
			"host.hostname": {hostname},
		},
	}
	config := struct {
		Duration int `json:"duration"`
		CpuLoad  int `json:"cpuLoad"`
		Workers  int `json:"workers"`
	}{Duration: 50000, Workers: 0, CpuLoad: 50}
	exec, err := e.RunAction("com.github.steadybit.extension_host.host.stress-cpu", target, config)
	require.NoError(t, err)

	assertProcessRunningInContainer(t, m, e.podName, "extension-host", "stress-ng")
	require.NoError(t, exec.Cancel())
}

func testStressMemory(t *testing.T, m *Minikube, e *Extension) {

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
		return
	}
	target := action_kit_api.Target{
		Attributes: map[string][]string{
			"host.hostname": {hostname},
		},
	}
	config := struct {
		Duration   int `json:"duration"`
		Percentage int `json:"percentage"`
	}{Duration: 50000, Percentage: 50}

	exec, err := e.RunAction("com.github.steadybit.extension_host.host.stress-mem", target, config)
	require.NoError(t, err)
	assertProcessRunningInContainer(t, m, e.podName, "extension-host", "stress-ng")
	require.NoError(t, exec.Cancel())
}

func testStressIo(t *testing.T, m *Minikube, e *Extension) {

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
		return
	}
	target := action_kit_api.Target{
		Attributes: map[string][]string{
			"host.hostname": {hostname},
		},
	}
	config := struct {
		Duration   int `json:"duration"`
		Percentage int `json:"percentage"`
		Workers    int `json:"workers"`
	}{Duration: 50000, Workers: 1, Percentage: 50}
	exec, err := e.RunAction("com.github.steadybit.extension_host.host.stress-io", target, config)
	require.NoError(t, err)
	assertProcessRunningInContainer(t, m, e.podName, "extension-host", "stress-ng")
	require.NoError(t, exec.Cancel())
}

func testTimeTravel(t *testing.T, m *Minikube, e *Extension) {

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
		return
	}
	target := action_kit_api.Target{
		Attributes: map[string][]string{
			"host.hostname": {hostname},
		},
	}
	config := struct {
		Duration   int  `json:"duration"`
		Offset     int  `json:"offset"`
		DisableNtp bool `json:"disableNtp"`
	}{Duration: 3000, Offset: 360000, DisableNtp: true}
	tolerance := time.Duration(1) * time.Second
	now := time.Now()
	exec, err := e.RunAction("com.github.steadybit.extension_host.timetravel", target, config)
	require.NoError(t, err)
	diff := getTimeDiffBetweenNowAndContainerTime(t, m, e, now)
	log.Debug().Msgf("diff: %s", diff)
	// check if is greater than offset

	assert.True(t, diff+tolerance > time.Duration(config.Offset)*time.Millisecond, "time travel failed")

	time.Sleep(3 * time.Second) // wait for rollback
	now = time.Now()
	diff = getTimeDiffBetweenNowAndContainerTime(t, m, e, now)
	log.Debug().Msgf("diff: %s", diff)
	assert.True(t, diff+tolerance <= 1*time.Second, "time travel failed to rollback properly")

	require.NoError(t, exec.Cancel())
}

func getTimeDiffBetweenNowAndContainerTime(t *testing.T, m *Minikube, e *Extension, now time.Time) time.Duration {
	out, err := getOutputOfCommand(m, e.podName, "extension-host", []string{"date", "+%s"})
	if err != nil {
		t.Fatal(err)
		return 0
	}
	containerSecondsSinceEpoch := exthost.ToInt64(strings.TrimSpace(out))
	if containerSecondsSinceEpoch == 0 {
		t.Fatal("could not parse container time")
		return 0
	}
	containerTime := time.Unix(containerSecondsSinceEpoch, 0)
	return containerTime.Sub(now)
}

func testDiscovery(t *testing.T, m *Minikube, e *Extension) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	target, err := pollForTarget(ctx, e, func(target discovery_kit_api.Target) bool {
		return hasAttribute(target, "host.hostname", "e2e-docker")
	})

	require.NoError(t, err)
	assert.Equal(t, target.TargetType, "host")
}
