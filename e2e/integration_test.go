// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-host/exthost"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

var (
	executionContext = &action_kit_api.ExecutionContext{
		AgentAwsAccountId: nil,
		RestrictedEndpoints: extutil.Ptr([]action_kit_api.RestrictedEndpoint{
			{
				Name:    "minikube ssh",
				Url:     "",
				Cidr:    "0.0.0.0/0",
				PortMin: 22,
				PortMax: 22,
			},
			{
				Name:    "minikube ssh",
				Url:     "",
				Cidr:    "::/0",
				PortMin: 22,
				PortMax: 22,
			},
			{
				Name:    "minikube k8s api",
				Url:     "",
				Cidr:    "0.0.0.0/0",
				PortMin: 8443,
				PortMax: 8443,
			},
			{
				Name:    "minikube k8s api",
				Url:     "",
				Cidr:    "::/0",
				PortMin: 8443,
				PortMax: 8443,
			},
		}),
	}
)

func getTarget(m *e2e.Minikube) *action_kit_api.Target {
	return &action_kit_api.Target{
		Attributes: map[string][]string{
			"host.hostname": {m.Profile},
		},
	}
}

func runsInCi() bool {
	return os.Getenv("CI") != ""
}
func TestWithMinikube(t *testing.T) {
	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-host",
		Port: 8085,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", fmt.Sprintf("container.runtime=%s", m.Runtime),
				"--set", "logging.level=debug",
			}
		},
	}

	mOpts := e2e.DefaultMiniKubeOpts
	if runtime.GOOS == "linux" {
		mOpts.Driver = "kvm2"
	}
	e2e.WithMinikube(t, mOpts, &extFactory, []e2e.WithMinikubeTestCase{
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
		//{
		//	Name: "shutdown host",
		//	Test: testShutdownHost, // if you run this test locally, you will need to restart your docker machine
		//},
		{
			Name: "network delay",
			Test: testNetworkDelay,
		},
		{
			Name: "network blackhole",
			Test: testNetworkBlackhole,
		},
		{
			Name: "network block dns",
			Test: testNetworkBlockDns,
		},
		{
			Name: "network limit bandwidth",
			Test: testNetworkLimitBandwidth,
		},
		{
			Name: "network package loss",
			Test: testNetworkPackageLoss,
		},
		{
			Name: "network package corruption",
			Test: testNetworkPackageCorruption,
		},
	})
}

func testStressCpu(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {

	config := struct {
		Duration int `json:"duration"`
		CpuLoad  int `json:"cpuLoad"`
		Workers  int `json:"workers"`
	}{Duration: 50000, Workers: 0, CpuLoad: 50}
	exec, err := e.RunAction("com.github.steadybit.extension_host.stress-cpu", getTarget(m), config, nil)
	require.NoError(t, err)

	e2e.AssertProcessRunningInContainer(t, m, e.Pod, "steadybit-extension-host", "stress-ng", true)
	require.NoError(t, exec.Cancel())
}

func testStressMemory(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {

	config := struct {
		Duration   int `json:"duration"`
		Percentage int `json:"percentage"`
	}{Duration: 50000, Percentage: 50}

	exec, err := e.RunAction("com.github.steadybit.extension_host.stress-mem", getTarget(m), config, nil)
	require.NoError(t, err)
	e2e.AssertProcessRunningInContainer(t, m, e.Pod, "steadybit-extension-host", "stress-ng", true)
	require.NoError(t, exec.Cancel())
}

func testStressIo(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {

	config := struct {
		Duration   int `json:"duration"`
		Percentage int `json:"percentage"`
		Workers    int `json:"workers"`
	}{Duration: 50000, Workers: 1, Percentage: 50}
	exec, err := e.RunAction("com.github.steadybit.extension_host.stress-io", getTarget(m), config, nil)
	require.NoError(t, err)
	e2e.AssertProcessRunningInContainer(t, m, e.Pod, "steadybit-extension-host", "stress-ng", true)
	require.NoError(t, exec.Cancel())
}

func testTimeTravel(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {

	config := struct {
		Duration   int  `json:"duration"`
		Offset     int  `json:"offset"`
		DisableNtp bool `json:"disableNtp"`
	}{Duration: 3000, Offset: 360000, DisableNtp: false}
	tolerance := time.Duration(1) * time.Second
	now := time.Now()
	exec, err := e.RunAction("com.github.steadybit.extension_host.timetravel", getTarget(m), config, nil)
	//if !runsInCi() { // time travel is not working in CI
	require.NoError(t, err)
	diff := getTimeDiffBetweenNowAndContainerTime(t, m, e, now)
	log.Debug().Msgf("diff: %s", diff)
	// check if is greater than offset

	assert.True(t, diff+tolerance > time.Duration(config.Offset)*time.Millisecond, "time travel failed")

	time.Sleep(3 * time.Second) // wait for rollback
	now = time.Now()
	diff = getTimeDiffBetweenNowAndContainerTime(t, m, e, now)
	log.Debug().Msgf("diff: %s", diff)
	assert.True(t, diff+tolerance <= 2*time.Second, "time travel failed to rollback properly")
	//}

	require.NoError(t, exec.Cancel())
}

func getTimeDiffBetweenNowAndContainerTime(t *testing.T, m *e2e.Minikube, e *e2e.Extension, now time.Time) time.Duration {
	out, err := m.Exec(e.Pod, "steadybit-extension-host", "date", "+%s")
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

func testDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	target, err := e2e.PollForTarget(ctx, e, "host", func(target discovery_kit_api.Target) bool {
		log.Debug().Msgf("targetHost: %v", target.Attributes["host.hostname"])
		return e2e.HasAttribute(target, "host.hostname", "e2e-docker")
	})

	require.NoError(t, err)
	assert.Equal(t, target.TargetType, "host")
}

func testStopProcess(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {

	config := struct {
		Duration int    `json:"duration"`
		Graceful bool   `json:"graceful"`
		Process  string `json:"process"`
		Delay    int    `json:"delay"`
	}{Duration: 10000, Graceful: true, Process: "tail", Delay: 1}

	e2e.AssertProcessNOTRunningInContainer(t, m, e.Pod, "steadybit-extension-host", "tail")
	go func() {
		_, _ = m.Exec(e.Pod, "steadybit-extension-host", "tail", "-f", "/dev/null")
	}()

	e2e.AssertProcessRunningInContainer(t, m, e.Pod, "steadybit-extension-host", "tail", true)

	exec, err := e.RunAction("com.github.steadybit.extension_host.stop-process", getTarget(m), config, nil)
	require.NoError(t, err)
	e2e.AssertProcessNOTRunningInContainer(t, m, e.Pod, "steadybit-extension-host", "tail")
	require.NoError(t, exec.Cancel())
}
func testShutdownHost(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	config := struct {
		Reboot bool `json:"reboot"`
	}{Reboot: true}

	exec, err := e.RunAction("com.github.steadybit.extension_host.shutdown", getTarget(m), config, nil)
	if !runsInCi() {
		require.NoError(t, err)
	}
	require.NoError(t, exec.Cancel())
}

func testNetworkBlackhole(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	nginx := e2e.Nginx{Minikube: m}
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
		{
			name:             "should blackhole only traffic for steadybit.com",
			hostname:         []string{"steadybit.com"},
			WantedReachable:  true,
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
			Duration: 30000,
			Ip:       tt.ip,
			Hostname: tt.hostname,
			Port:     tt.port,
		}

		t.Run(tt.name, func(t *testing.T) {
			nginx.AssertIsReachable(t, true)
			nginx.AssertCanReach(t, "https://steadybit.com", true)

			action, err := e.RunAction(exthost.BaseActionID+".network_blackhole", getTarget(m), config, executionContext)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			nginx.AssertIsReachable(t, tt.WantedReachable)
			nginx.AssertCanReach(t, "https://steadybit.com", tt.WantedReachesUrl)

			require.NoError(t, action.Cancel())
			nginx.AssertIsReachable(t, true)
			nginx.AssertCanReach(t, "https://steadybit.com", true)
		})
	}
}

func testNetworkDelay(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	netperf := e2e.Netperf{Minikube: m}
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

		t.Run(tt.name, func(t *testing.T) {
			action, err := e.RunAction(exthost.BaseActionID+".network_delay", getTarget(m), config, executionContext)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			if tt.WantedDelay {
				netperf.AssertLatency(t, unaffectedLatency+time.Duration(config.Delay)*time.Millisecond*90/100, unaffectedLatency+time.Duration(config.Delay)*time.Millisecond*350/100)
			} else {
				netperf.AssertLatency(t, 0, unaffectedLatency*110/100)
			}
			require.NoError(t, action.Cancel())

			netperf.AssertLatency(t, 0, unaffectedLatency*110/100)
		})
	}
}

func testNetworkPackageLoss(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	iperf := e2e.Iperf{Minikube: m}
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
			Percentage   int      `json:"percentage"`
			Ip           []string `json:"ip"`
			Hostname     []string `json:"hostname"`
			Port         []string `json:"port"`
			NetInterface []string `json:"networkInterface"`
		}{
			Duration:     10000,
			Percentage:   10,
			Ip:           tt.ip,
			Hostname:     tt.hostname,
			Port:         tt.port,
			NetInterface: tt.interfaces,
		}

		t.Run(tt.name, func(t *testing.T) {
			action, err := e.RunAction(exthost.BaseActionID+".network_package_loss", getTarget(m), config, executionContext)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			if tt.WantedLoss {
				iperf.AssertPackageLoss(t, float64(config.Percentage)*0.8, float64(config.Percentage)*1.2)
			} else {
				iperf.AssertPackageLoss(t, 0, 5)
			}
			require.NoError(t, action.Cancel())

			iperf.AssertPackageLoss(t, 0, 5)
		})
	}
}

func testNetworkPackageCorruption(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	iperf := e2e.Iperf{Minikube: m}
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

		t.Run(tt.name, func(t *testing.T) {
			action, err := e.RunAction(exthost.BaseActionID+".network_package_corruption", getTarget(m), config, executionContext)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			if tt.WantedCorruption {
				iperf.AssertPackageLoss(t, float64(config.Corruption)*0.8, float64(config.Corruption)*1.2)
			} else {
				iperf.AssertPackageLoss(t, 0, 5)
			}
			require.NoError(t, action.Cancel())

			iperf.AssertPackageLoss(t, 0, 5)
		})
	}
}

func testNetworkLimitBandwidth(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	iperf := e2e.Iperf{Minikube: m}
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
			name: "should limit bandwidth only on port 5001 traffic",
			port: []string{"5001"},
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
	limited := unlimited / 3

	for _, tt := range tests {
		config := struct {
			Duration     int      `json:"duration"`
			Bandwidth    string   `json:"bandwidth"`
			Ip           []string `json:"ip"`
			Hostname     []string `json:"hostname"`
			Port         []string `json:"port"`
			NetInterface []string `json:"networkInterface"`
		}{
			Duration:     30000,
			Bandwidth:    fmt.Sprintf("%dmbit", int(limited)),
			Ip:           tt.ip,
			Hostname:     tt.hostname,
			Port:         tt.port,
			NetInterface: tt.interfaces,
		}

		t.Run(tt.name, func(t *testing.T) {
			action, err := e.RunAction(exthost.BaseActionID+".network_bandwidth", getTarget(m), config, executionContext)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			if tt.WantedLimit {
				iperf.AssertBandwidth(t, limited*0.95, limited*1.05)
			} else {
				iperf.AssertBandwidth(t, unlimited*0.95, unlimited*1.05)
			}
			require.NoError(t, action.Cancel())
			iperf.AssertBandwidth(t, unlimited*0.95, unlimited*1.05)
		})
	}
}

func testNetworkBlockDns(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	nginx := e2e.Nginx{Minikube: m}
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

		t.Run(tt.name, func(t *testing.T) {
			nginx.AssertIsReachable(t, true)
			nginx.AssertCanReach(t, "https://steadybit.com", true)

			action, err := e.RunAction(exthost.BaseActionID+".network_block_dns", getTarget(m), config, executionContext)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)

			nginx.AssertIsReachable(t, tt.WantedReachable)
			if tt.WantedReachesUrl {
				nginx.AssertCanReach(t, "https://steadybit.com", true)
			} else {
				nginx.AssertCannotReach(t, "https://steadybit.com", "Resolving timed out after")
			}

			require.NoError(t, action.Cancel())
			nginx.AssertIsReachable(t, true)
			nginx.AssertCanReach(t, "https://steadybit.com", true)
		})
	}
}
