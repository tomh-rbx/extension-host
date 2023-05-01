// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"context"
	"fmt"
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
	executionContext = &action_kit_api.ExecutionContext{
		AgentAwsAccountId: nil,
		RestrictedUrls:    extutil.Ptr([]string{"http://0.0.0.0:8443", "http://0.0.0.0:8085"}),
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
		//{
		//	Name: "network delay",
		//	Test: testNetworkDelay,
		//},
		//{
		//	Name: "network block dns",
		//	Test: testNetworkBlockDns,
		//}, {
		//	Name: "network limit bandwidth",
		//	Test: testNetworkLimitBandwidth,
		//},
		//{
		//	Name: "network package loss",
		//	Test: testNetworkPackageLoss,
		//},
		//{
		//	Name: "network package corruption",
		//	Test: testNetworkPackageCorruption,
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

	assertProcessRunningInContainer(t, m, e.pod, "steadybit-extension-host", "stress-ng")
	require.NoError(t, exec.Cancel())
}

func testStressMemory(t *testing.T, m *Minikube, e *Extension) {

	config := struct {
		Duration   int `json:"duration"`
		Percentage int `json:"percentage"`
	}{Duration: 50000, Percentage: 50}

	exec, err := e.RunAction("com.github.steadybit.extension_host.stress-mem", target, config, nil)
	require.NoError(t, err)
	assertProcessRunningInContainer(t, m, e.pod, "steadybit-extension-host", "stress-ng")
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
	assertProcessRunningInContainer(t, m, e.pod, "steadybit-extension-host", "stress-ng")
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
	out, err := m.Exec(e.pod, "steadybit-extension-host", "date", "+%s")
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
	}{Duration: 10000, Graceful: true, Process: "tail", Delay: 1}

	assertProcessNOTRunningInContainer(t, m, e.pod, "steadybit-extension-host", "tail")
	go func() {
		_, _ = m.Exec(e.pod, "steadybit-extension-host", "tail", "-f", "/dev/null")
	}()

	assertProcessRunningInContainer(t, m, e.pod, "steadybit-extension-host", "tail")

	exec, err := e.RunAction("com.github.steadybit.extension_host.stop-process", target, config, nil)
	require.NoError(t, err)
	assertProcessNOTRunningInContainer(t, m, e.pod, "steadybit-extension-host", "tail")
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

func testNetworkDelay(t *testing.T, m *Minikube, e *Extension) {
	netperf := netperf{minikube: m}
	err := netperf.Deploy("delay")
	defer func() { _ = netperf.Delete() }()
	require.NoError(t, err)

	tests := []struct {
		name        string
		ip          []string
		hostname    []string
		port        []string
		interfaces  []string
		WantedDelay bool
	}{
		{
			name:        "should delay all traffic",
			WantedDelay: true,
		},
		{
			name:        "should delay only port 5000 traffic",
			port:        []string{"5000"},
			interfaces:  []string{"eth0"},
			WantedDelay: true,
		},
		{
			name:        "should delay only port 80 traffic",
			port:        []string{"80"},
			WantedDelay: false,
		},
	}

	unaffectedLatency, err := netperf.MeasureLatency()
	require.NoError(t, err)

	for _, tt := range tests {
		config := struct {
			Duration     int      `json:"duration"`
			Delay        int      `json:"networkDelay"`
			Jitter       bool     `json:"networkDelayJitter"`
			Ip           []string `json:"ip"`
			Hostname     []string `json:"hostname"`
			Port         []string `json:"port"`
			NetInterface []string `json:"networkInterface"`
		}{
			Duration:     10000,
			Delay:        200,
			Jitter:       false,
			Ip:           tt.ip,
			Hostname:     tt.hostname,
			Port:         tt.port,
			NetInterface: tt.interfaces,
		}

		if m.stdout == nil {
			m.stdout = os.Stdout
		}
		t.Run(tt.name, func(t *testing.T) {
			action, err := e.RunAction(exthost.BaseActionID+".network_delay", target, config, executionContext)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			latency, err := netperf.MeasureLatency()
			require.NoError(t, err)
			delay := latency - unaffectedLatency
			if tt.WantedDelay {
				require.True(t, delay > 200*time.Millisecond, "service should be delayed >200ms but was delayed %s", delay.String())
			} else {
				require.True(t, delay < 50*time.Millisecond, "service should not be delayed but was delayed %s", delay.String())
			}
			require.NoError(t, action.Cancel())

			latency, err = netperf.MeasureLatency()
			require.NoError(t, err)
			delay = latency - unaffectedLatency
			require.True(t, delay < 50*time.Millisecond, "service should not be delayed but was delayed %s", delay.String())
		})
	}
}

func testNetworkPackageLoss(t *testing.T, m *Minikube, e *Extension) {
	iperf := iperf{minikube: m}
	err := iperf.Deploy("loss")
	defer func() { _ = iperf.Delete() }()
	require.NoError(t, err)

	tests := []struct {
		name       string
		ip         []string
		hostname   []string
		port       []string
		interfaces []string
		WantedLoss bool
	}{
		{
			name:       "should loose packages on all traffic",
			WantedLoss: true,
		},
		{
			name:       "should loose packages only on port 5001 traffic",
			port:       []string{"5001"},
			interfaces: []string{"eth0"},
			WantedLoss: true,
		},
		{
			name:       "should loose packages only on port 80 traffic",
			port:       []string{"80"},
			WantedLoss: false,
		},
	}

	for _, tt := range tests {
		config := struct {
			Duration     int      `json:"duration"`
			Loss         int      `json:"networkLoss"`
			Ip           []string `json:"ip"`
			Hostname     []string `json:"hostname"`
			Port         []string `json:"port"`
			NetInterface []string `json:"networkInterface"`
		}{
			Duration:     10000,
			Loss:         10,
			Ip:           tt.ip,
			Hostname:     tt.hostname,
			Port:         tt.port,
			NetInterface: tt.interfaces,
		}

		if m.stdout == nil {
			m.stdout = os.Stdout
		}
		t.Run(tt.name, func(t *testing.T) {
			action, err := e.RunAction(exthost.BaseActionID+".network_package_loss", target, config, executionContext)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			loss, err := iperf.MeasurePackageLoss()
			require.NoError(t, err)
			if tt.WantedLoss {
				require.True(t, loss >= 7.0, "~10%% packages should be lost but was %.2f", loss)
			} else {
				require.True(t, loss <= 2.0, "packages should be lost but was %.2f", loss)
			}
			require.NoError(t, action.Cancel())

			loss, err = iperf.MeasurePackageLoss()
			require.NoError(t, err)
			require.True(t, loss <= 2.0, "packages should be lost but was %.2f", loss)
		})
	}
}

func testNetworkPackageCorruption(t *testing.T, m *Minikube, e *Extension) {
	iperf := iperf{minikube: m}
	err := iperf.Deploy("corruption")
	defer func() { _ = iperf.Delete() }()
	require.NoError(t, err)

	tests := []struct {
		name             string
		ip               []string
		hostname         []string
		port             []string
		interfaces       []string
		WantedCorruption bool
	}{
		{
			name:             "should corrupt packages on all traffic",
			WantedCorruption: true,
		},
		{
			name:             "should corrupt packages only on port 5001 traffic",
			port:             []string{"5001"},
			interfaces:       []string{"eth0"},
			WantedCorruption: true,
		},
		{
			name:             "should corrupt packages only on port 80 traffic",
			port:             []string{"80"},
			WantedCorruption: false,
		},
	}

	for _, tt := range tests {
		config := struct {
			Duration     int      `json:"duration"`
			Corruption   int      `json:"networkCorruption"`
			Ip           []string `json:"ip"`
			Hostname     []string `json:"hostname"`
			Port         []string `json:"port"`
			NetInterface []string `json:"networkInterface"`
		}{
			Duration:     10000,
			Corruption:   10,
			Ip:           tt.ip,
			Hostname:     tt.hostname,
			Port:         tt.port,
			NetInterface: tt.interfaces,
		}

		if m.stdout == nil {
			m.stdout = os.Stdout
		}
		t.Run(tt.name, func(t *testing.T) {
			action, err := e.RunAction(exthost.BaseActionID+".network_package_corruption", target, config, executionContext)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			loss, err := iperf.MeasurePackageLoss()
			require.NoError(t, err)
			if tt.WantedCorruption {
				require.True(t, loss >= 7.0, "~10%% packages should be corrupted but was %.2f", loss)
			} else {
				require.True(t, loss <= 2.0, "packages should be corrupted but was %.2f", loss)
			}
			require.NoError(t, action.Cancel())

			loss, err = iperf.MeasurePackageLoss()
			require.NoError(t, err)
			require.True(t, loss <= 2.0, "packages should be corrupted but was %.2f", loss)
		})
	}
}

func testNetworkLimitBandwidth(t *testing.T, m *Minikube, e *Extension) {
	iperf := iperf{minikube: m}
	err := iperf.Deploy("bandwidth")
	defer func() { _ = iperf.Delete() }()
	require.NoError(t, err)

	tests := []struct {
		name        string
		ip          []string
		hostname    []string
		port        []string
		interfaces  []string
		WantedLimit bool
	}{
		{
			name:        "should limit bandwidth on all traffic",
			WantedLimit: true,
		},
		{
			name:        "should limit bandwidth only on port 5001 traffic",
			port:        []string{"5001"},
			interfaces:  []string{"eth0"},
			WantedLimit: true,
		},
		{
			name:        "should limit bandwidth only on port 80 traffic",
			port:        []string{"80"},
			WantedLimit: false,
		},
	}

	unlimited, err := iperf.MeasureBandwidth()
	require.NoError(t, err)
	limit := unlimited / 3

	for _, tt := range tests {
		config := struct {
			Duration     int      `json:"duration"`
			Bandwidth    string   `json:"bandwidth"`
			Ip           []string `json:"ip"`
			Hostname     []string `json:"hostname"`
			Port         []string `json:"port"`
			NetInterface []string `json:"networkInterface"`
		}{
			Duration:     10000,
			Bandwidth:    fmt.Sprintf("%dmbit", int(limit)),
			Ip:           tt.ip,
			Hostname:     tt.hostname,
			Port:         tt.port,
			NetInterface: tt.interfaces,
		}

		if m.stdout == nil {
			m.stdout = os.Stdout
		}
		t.Run(tt.name, func(t *testing.T) {
			action, err := e.RunAction(exthost.BaseActionID+".network_bandwidth", target, config, executionContext)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			bandwidth, err := iperf.MeasureBandwidth()
			require.NoError(t, err)
			if tt.WantedLimit {
				require.True(t, bandwidth <= (limit*1.05), "bandwidth should be ~%.2fmbit but was %.2fmbit", limit, bandwidth)
			} else {
				require.True(t, bandwidth > (unlimited*0.95), "bandwidth should not be limited (~%.2fmbit) but was %.2fmbit", unlimited, bandwidth)
			}
			require.NoError(t, action.Cancel())

			bandwidth, err = iperf.MeasureBandwidth()
			require.NoError(t, err)
			require.True(t, bandwidth > (unlimited*0.95), "bandwidth should not be limited (~%.2fmbit) but was %.2fmbit", unlimited, bandwidth)
		})
	}
}

func testNetworkBlockDns(t *testing.T, m *Minikube, e *Extension) {
	nginx := Nginx{minikube: m}
	err := nginx.Deploy("nginx-network-block-dns")
	require.NoError(t, err, "failed to create pod")
	defer func() { _ = nginx.Delete() }()

	tests := []struct {
		name             string
		ip               []string
		hostname         []string
		dnsPort          uint
		WantedReachable  bool
		WantedReachesUrl bool
	}{
		{
			name:             "should block dns traffic",
			dnsPort:          53,
			WantedReachable:  true,
			WantedReachesUrl: false,
		},
		{
			name:             "should block dns traffic on port 5353",
			dnsPort:          5353,
			WantedReachable:  true,
			WantedReachesUrl: true,
		},
	}

	for _, tt := range tests {
		config := struct {
			Duration int  `json:"duration"`
			DnsPort  uint `json:"dnsPort"`
		}{
			Duration: 10000,
			DnsPort:  tt.dnsPort,
		}

		if m.stdout == nil {
			m.stdout = os.Stdout
		}
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, nginx.IsReachable(), "service should be reachable before block dns")
			require.NoError(t, nginx.CanReach("https://google.com"), "service should reach url before block dns")

			action, err := e.RunAction(exthost.BaseActionID+".network_block_dns", target, config, executionContext)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			if tt.WantedReachable {
				require.NoError(t, nginx.IsReachable(), "service should be reachable during block dns")
			} else {
				require.Error(t, nginx.IsReachable(), "service should not be reachable during block dns")
			}

			if tt.WantedReachesUrl {
				require.NoError(t, nginx.CanReach("https://google.com"), "service should be reachable during block dns")
			} else {
				require.ErrorContains(t, nginx.CanReach("https://google.com"), "Resolving timed out", "service should not be reachable during block dns")
			}

			require.NoError(t, action.Cancel())
			require.NoError(t, nginx.IsReachable(), "service should be reachable after block dns")
			require.NoError(t, nginx.CanReach("https://google.com"), "service should reach url after block dns")
		})
	}
}
