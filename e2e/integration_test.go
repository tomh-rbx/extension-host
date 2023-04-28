// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-host/exthost"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	target = action_kit_api.Target{
		Attributes: map[string][]string{
			"host.hostname": {"e2e-docker"},
		},
	}
)

func runsInCi() bool {
	return os.Getenv("CI") != ""
}
func TestWithMinikube(t *testing.T) {
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
			Name: "time travel",
			Test: testTimeTravel,
		},
		{
			Name: "stop process",
			Test: testStopProcess,
		},
		{
			Name: "shutdown host",
			Test: testShutdownHost,
		},
		//{
		//	Name: "network blackhole",
		//	Test: testNetworkBlackhole,
		//},
	})
}

func testStressCpu(t *testing.T, m *Minikube, e *Extension) {

	config := struct {
		Duration int `json:"duration"`
		CpuLoad  int `json:"cpuLoad"`
		Workers  int `json:"workers"`
	}{Duration: 50000, Workers: 0, CpuLoad: 50}
	exec, err := e.RunAction("com.github.steadybit.extension_host.stress-cpu", target, config, nil)
	require.NoError(t, err)

	assertProcessRunningInContainer(t, m, e.pod, "extension-host", "stress-ng")
	require.NoError(t, exec.Cancel())
}

func testStressMemory(t *testing.T, m *Minikube, e *Extension) {

	config := struct {
		Duration   int `json:"duration"`
		Percentage int `json:"percentage"`
	}{Duration: 50000, Percentage: 50}

	exec, err := e.RunAction("com.github.steadybit.extension_host.stress-mem", target, config, nil)
	require.NoError(t, err)
	assertProcessRunningInContainer(t, m, e.pod, "extension-host", "stress-ng")
	require.NoError(t, exec.Cancel())
}

func testStressIo(t *testing.T, m *Minikube, e *Extension) {

	config := struct {
		Duration   int `json:"duration"`
		Percentage int `json:"percentage"`
		Workers    int `json:"workers"`
	}{Duration: 50000, Workers: 1, Percentage: 50}
	exec, err := e.RunAction("com.github.steadybit.extension_host.stress-io", target, config, nil)
	require.NoError(t, err)
	assertProcessRunningInContainer(t, m, e.pod, "extension-host", "stress-ng")
	require.NoError(t, exec.Cancel())
}

func testTimeTravel(t *testing.T, m *Minikube, e *Extension) {

	config := struct {
		Duration   int  `json:"duration"`
		Offset     int  `json:"offset"`
		DisableNtp bool `json:"disableNtp"`
	}{Duration: 3000, Offset: 360000, DisableNtp: false}
	tolerance := time.Duration(1) * time.Second
	now := time.Now()
	exec, err := e.RunAction("com.github.steadybit.extension_host.timetravel", target, config, nil)
	if !runsInCi() {
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
	}

	require.NoError(t, exec.Cancel())
}

func getTimeDiffBetweenNowAndContainerTime(t *testing.T, m *Minikube, e *Extension, now time.Time) time.Duration {
	out, err := m.Exec(e.pod, "extension-host", "date", "+%s")
	if err != nil {
		t.Fatal(err)
		return 0
	}
	containerSecondsSinceEpoch := extutil.ToInt64(strings.TrimSpace(out))
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
		log.Debug().Msgf("targetHost: %v", target.Attributes["host.hostname"])
		return hasAttribute(target, "host.hostname", "e2e-docker")
	})

	require.NoError(t, err)
	assert.Equal(t, target.TargetType, "host")
}

func testStopProcess(t *testing.T, m *Minikube, e *Extension) {

	config := struct {
		Duration int    `json:"duration"`
		Graceful bool   `json:"graceful"`
		Process  string `json:"process"`
		Delay    int    `json:"delay"`
	}{Duration: 10000, Graceful: true, Process: "sleep", Delay: 1}

	assertProcessNOTRunningInContainer(t, m, e.pod, "extension-host", "sleep")
	go func() {
		_, _ = m.Exec(e.pod, "extension-host", "sleep", "30")
	}()

	assertProcessRunningInContainer(t, m, e.pod, "extension-host", "sleep")

	exec, err := e.RunAction("com.github.steadybit.extension_host.stop-process", target, config, nil)
	require.NoError(t, err)
	assertProcessNOTRunningInContainer(t, m, e.pod, "extension-host", "sleep")
	require.NoError(t, exec.Cancel())
}
func testShutdownHost(t *testing.T, m *Minikube, e *Extension) {

	config := struct {
		Reboot bool `json:"reboot"`
	}{Reboot: true}

	exec, err := e.RunAction("com.github.steadybit.extension_host.shutdown", target, config, nil)
	if !runsInCi() {
		require.NoError(t, err)
	}
	require.NoError(t, exec.Cancel())
}

func testNetworkBlackhole(t *testing.T, m *Minikube, e *Extension) {
	executionContext := &action_kit_api.ExecutionContext{
		AgentAwsAccountId: nil,
		RestrictedUrls:    extutil.Ptr([]string{"http://0.0.0.0:8443", "http://0.0.0.0:8085"}),
	}
	nginx := Nginx{minikube: m}
	err := nginx.Deploy("nginx-network-blackhole")
	require.NoError(t, err, "failed to create pod")
	defer func() { _ = nginx.Delete() }()

	tests := []struct {
		name             string
		ip               []string
		hostname         []string
		port             []string
		WantedReachable  bool
		WantedReachesUrl bool
	}{
		{
			name:             "should blackhole all traffic",
			WantedReachable:  false,
			WantedReachesUrl: false,
		},
		{
			name:             "should blackhole only port 8080 traffic",
			port:             []string{"8080"},
			WantedReachable:  true,
			WantedReachesUrl: true,
		},
		{
			name:             "should blackhole only port 80, 443 traffic",
			port:             []string{"80", "443"},
			WantedReachable:  false,
			WantedReachesUrl: false,
		},
	}

	for _, tt := range tests {
		config := struct {
			Duration int      `json:"duration"`
			Ip       []string `json:"ip"`
			Hostname []string `json:"hostname"`
			Port     []string `json:"port"`
		}{
			Duration: 10000,
			Ip:       tt.ip,
			Hostname: tt.hostname,
			Port:     tt.port,
		}

		if m.stdout == nil {
			m.stdout = os.Stdout
		}

		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, nginx.IsReachable(), "service should be reachable before blackhole")
			require.NoError(t, nginx.CanReach("https://google.com"), "service should reach url before blackhole")

			action, err := e.RunAction(exthost.BaseActionID+".network_blackhole", target, config, executionContext)
			require.NoError(t, err)

			if tt.WantedReachable {
				require.NoError(t, nginx.IsReachable(), "service should be reachable during blackhole")
			} else {
				require.Error(t, nginx.IsReachable(), "service should not be reachable during blackhole")
			}

			if tt.WantedReachesUrl {
				require.NoError(t, nginx.CanReach("https://google.com"), "service should be reachable during blackhole")
			} else {
				require.Error(t, nginx.CanReach("https://google.com"), "service should not be reachable during blackhole")
			}

			require.NoError(t, action.Cancel())
			require.NoError(t, nginx.IsReachable(), "service should be reachable after blackhole")
			require.NoError(t, nginx.CanReach("https://google.com"), "service should reach url after blackhole")
		})
	}
}
