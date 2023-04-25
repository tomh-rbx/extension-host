// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
  "bufio"
  "context"
  "encoding/json"
  "errors"
  "fmt"
  "github.com/go-resty/resty/v2"
  "github.com/rs/zerolog"
  "github.com/rs/zerolog/log"
  "github.com/stretchr/testify/require"
  "golang.org/x/exp/slices"
  "io"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/client-go/kubernetes"
  "k8s.io/client-go/rest"
  "k8s.io/client-go/tools/clientcmd"
  "k8s.io/client-go/util/homedir"
  "os"
  "os/exec"
  "path/filepath"
  "strings"
  "sync"
  "testing"
  "time"
)

var (
  globalMinikubeMutex sync.Mutex
)

type Minikube struct {
  profile string
  stdout  io.Writer
  stderr  io.Writer

  clientOnce   sync.Once
  client       *kubernetes.Clientset
  clientConfig *rest.Config
}

type Config struct {
  Name         string
  ExposedPorts []string
}

type Profile struct {
  Name   string
  Status string
  Config Config
}

type Profiles struct {
  Valid   []Profile `json:"valid"`
  Invalid []Profile `json:"invalid"`
}

func checkMinikubeIsRunning() string {

  cmd := exec.Command("minikube", "profile", "list", "--output", "json")
  stdout, err := cmd.Output()
  if err != nil {
    log.Error().Err(err).Msg("failed to get the profile list")
    return ""
  }

  profiles := Profiles{}
  if err := json.Unmarshal(stdout, &profiles); err != nil {
    log.Error().Err(err).Msg("failed to unmarshal the profile list")
    return ""
  }
  for _, profile := range profiles.Valid {
    if profile.Status == "Running" && slices.Contains(profile.Config.ExposedPorts, "8085") {
      return profile.Name
    }
  }
  return ""
}

func newMinikube() *Minikube {
  runtime := "docker"
  profile := "e2e-" + runtime
  stdout := prefixWriter{prefix: "🧊", w: os.Stdout}
  stderr := prefixWriter{prefix: "🧊", w: os.Stderr}

  return &Minikube{
    profile: profile,
    stdout:  &stdout,
    stderr:  &stderr,
  }
}
func (m *Minikube) start() error {
  globalMinikubeMutex.Lock()
  defer globalMinikubeMutex.Unlock()

  args := []string{"start", "--keep-context", "--container-runtime=docker", "--ports=8085"}

  if err := m.command(args...).Run(); err != nil {
    return err
  }

  return nil
}

func (m *Minikube) Client() *kubernetes.Clientset {
  if m.client == nil {
    m.clientOnce.Do(func() {
      client, config, err := createKubernetesClient(m.profile)
      if err != nil {
        log.Fatal().Err(err).Msg("failed to create kubernetes client")
      }
      m.client = client
      m.clientConfig = config
    })
  }
  return m.client
}

func (m *Minikube) Config() *rest.Config {
  if m.clientConfig == nil {
    m.Client()
  }
  return m.clientConfig
}

func (m *Minikube) waitForDefaultServiceaccount() error {
  ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
  defer cancel()

  for {
    select {
    case <-ctx.Done():
      return errors.New("the serviceaccount 'default' was not created")
    case <-time.After(200 * time.Millisecond):
      if _, err := m.Client().CoreV1().ServiceAccounts("default").Get(context.Background(), "default", metav1.GetOptions{}); err == nil {
        return nil
      }
    }
  }
}

func (m *Minikube) delete() error {
  globalMinikubeMutex.Lock()
  defer globalMinikubeMutex.Unlock()
  return m.command("delete").Run()
}

func (m *Minikube) cp(src, dst string) error {
  return m.command(m.profile, "cp", src, dst).Run()
}

func (m *Minikube) exec(arg ...string) *exec.Cmd {
  return m.command(append([]string{"ssh", "--"}, arg...)...)
}

func (m *Minikube) LoadImage(image string) error {
  return m.command("image", "load", image).Run()
}

func (m *Minikube) command(arg ...string) *exec.Cmd {
  return m.commandContext(context.Background(), arg...)
}

func (m *Minikube) commandContext(ctx context.Context, arg ...string) *exec.Cmd {
  cmd := exec.CommandContext(ctx, "minikube", append([]string{"-p", m.profile}, arg...)...)
  cmd.Stdout = m.stdout
  cmd.Stderr = m.stderr
  return cmd
}

type WithMinikubeTestCase struct {
  Name string
  Test func(t *testing.T, minikube *Minikube, e *Extension)
}

func WithMinikube(t *testing.T, testCases []WithMinikubeTestCase) {
  log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

  imageName := ""
  wg := sync.WaitGroup{}
  wg.Add(1)
  go func() {
    s, err := createExtensionContainer()
    if err != nil {
      log.Error().Err(err).Msg("failed to create extension executable")
      panic(err)
    }
    imageName = s
    wg.Done()
  }()

  t.Run("docker", func(t *testing.T) {
    minikubeProfileName := checkMinikubeIsRunning()
    var minikube *Minikube
    var err error
    if minikubeProfileName == "" {
      minikube := newMinikube()
      _ = minikube.delete()
      err := minikube.start()
      if err != nil {
        t.Fatalf("failed to start minikube: %v", err)
      }
    } else {
      minikube = &Minikube{profile: minikubeProfileName}
    }
    defer func() {
      if minikubeProfileName == "" {
        _ = minikube.delete()
      }
    }()

    t.Parallel()

    if err := minikube.waitForDefaultServiceaccount(); err != nil {
      t.Fatal("Serviceaccount didn't show up", err)
    }

    wg.Wait()
    extension, err := startExtension(minikube, imageName)
    require.NoError(t, err)
    defer func() { _ = extension.stop() }()

    for _, tc := range testCases {
      t.Run(tc.Name, func(t *testing.T) {
        tc.Test(t, minikube, extension)
      })
    }
  })

}

func createKubernetesClient(context string) (*kubernetes.Clientset, *rest.Config, error) {
  config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
    &clientcmd.ClientConfigLoadingRules{ExplicitPath: filepath.Join(homedir.HomeDir(), ".kube", "config")},
    &clientcmd.ConfigOverrides{CurrentContext: context},
  ).ClientConfig()
  if err != nil {
    return nil, nil, err
  }

  client, err := kubernetes.NewForConfig(config)
  return client, config, err
}

type prefixWriter struct {
  prefix string
  w      io.Writer
}

func (w *prefixWriter) Write(p []byte) (n int, err error) {
  lines := strings.Split(strings.TrimSuffix(string(p), "\n"), "\n")
  count := 0
  for _, line := range lines {
    c, err := fmt.Fprintf(w.w, "%s%s\n", w.prefix, line)
    count += c
    if err != nil {
      return count, err
    }
  }
  return len(p), nil
}

type ServiceClient struct {
  resty.Client
  close func()
}

func (c *ServiceClient) Close() {
  c.close()
}

func (m *Minikube) NewServiceClient(service metav1.Object) (*ServiceClient, error) {
  url, cancel, err := m.tunnelService(service)
  if err != nil {
    return nil, err
  }

  client := resty.New()
  client.SetBaseURL(url)
  client.SetTimeout(3 * time.Second)

  return &ServiceClient{
    Client: *client,
    close:  cancel,
  }, nil
}

func (m *Minikube) tunnelService(service metav1.Object) (string, func(), error) {
  ctx, cancel := context.WithCancel(context.Background())
  cmd := m.commandContext(ctx, "service", "--namespace", service.GetNamespace(), service.GetName(), "--url")
  cmd.Stdout = nil
  stdout, err := cmd.StdoutPipe()
  if err != nil {
    cancel()
    return "", nil, err
  }

  chUrl := make(chan string)
  go func(r io.Reader) {
    scanner := bufio.NewScanner(r)
    for {
      if !scanner.Scan() {
        return
      }
      line := scanner.Text()
      _, _ = m.stdout.Write([]byte(line))
      if strings.HasPrefix(line, "http") {
        chUrl <- line
        return
      }
    }
  }(stdout)

  err = cmd.Start()
  if err != nil {
    cancel()
    return "", nil, err
  }

  chErr := make(chan error)
  go func() { chErr <- cmd.Wait() }()

  select {
  case url := <-chUrl:
    return url, cancel, nil
  case <-time.After(10 * time.Second):
    cancel()
    return "", nil, fmt.Errorf("timed out to tunnel service")
  case err = <-chErr:
    cancel()
    return "", nil, fmt.Errorf("failed to tunnel service: %w", err)
  }
}
