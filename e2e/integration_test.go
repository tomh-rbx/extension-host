// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
  "context"
  "github.com/steadybit/action-kit/go/action_kit_api/v2"
  "github.com/steadybit/discovery-kit/go/discovery_kit_api"
  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/require"
  "os"
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
  }{Duration: 1000, Workers: 0, CpuLoad: 50}
  exec := e.RunAction("com.github.steadybit.extension_host.host.stress-cpu", target, config)

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
  }{Duration: 5000, Percentage: 50}

  exec := e.RunAction("com.github.steadybit.extension_host.host.stress-mem", target, config)
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
  }{Duration: 5000, Workers: 1, Percentage: 50}
  exec := e.RunAction("com.github.steadybit.extension_host.host.stress-io", target, config)
  assertProcessRunningInContainer(t, m, e.podName, "extension-host", "stress-ng")
  require.NoError(t, exec.Cancel())
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
