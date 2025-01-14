<img src="./logo.svg" height="130" align="right" alt="Host logo">

# Steadybit extension-host

This [Steadybit](https://www.steadybit.com/) extension provides a host discovery and various actions for host targets.

Learn about the capabilities of this extension in our [Reliability Hub](https://hub.steadybit.com/extension/com.steadybit.extension_host).

## Configuration

| Environment Variable                                     | Helm value                         | Meaning                                                                                                                                                                                                                       | Required | Default |
|----------------------------------------------------------|------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `STEADYBIT_LABEL_<key>=<value>`                          |                                    | Environment variables starting with `STEADYBIT_LABEL_` will be added to discovered targets' attributes. <br>**Example:** `STEADYBIT_LABEL_TEAM=Fullfillment` adds to each discovered target the attribute `team=Fullfillment` | no       |         |
| `STEADYBIT_DISCOVERY_ENV_LIST`                           |                                    | List of environment variables to be evaluated and added to discovered targets' attributes. <br> **Example:** `STEADYBIT_DISCOVERY_ENV_LIST=STAGE` adds to each target the attribute `stage=<value of $STAGE>`                 | no       |         |
| `STEADYBIT_EXTENSION_DISCOVERY_ATTRIBUTES_EXCLUDES_HOST` | discovery.attributes.excludes.host | List of Target Attributes which will be excluded during discovery. Checked by key equality and supporting trailing "*"                                                                                                        | false    |         |

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

When installed as linux package this configuration is in`/etc/steadybit/extension-host`.

## Needed capabilities

The capabilities needed by this extension are: (which are provided by the helm chart)

- `SYS_ADMIN`
- `SYS_RESOURCE`
- `SYS_BOOT`
- `NET_RAW`
- `SYS_TIME`
- `SYS_PTRACE`
- `KILL`
- `NET_ADMIN`
- `DAC_OVERRIDE`
- `SETUID`
- `SETGID`
- `AUDIT_WRITE`

## Installation

### Kubernetes

Detailed information about agent and extension installation in kubernetes can also be found in
our [documentation](https://docs.steadybit.com/install-and-configure/install-agent/install-on-kubernetes).

#### Recommended (via agent helm chart)

All extensions provide a helm chart that is also integrated in the
[helm-chart](https://github.com/steadybit/helm-charts/tree/main/charts/steadybit-agent) of the agent.

The extension is installed by default when you install the agent.

You can provide additional values to configure this extension.

Additional configuration options can be found in
the [helm-chart](https://github.com/steadybit/extension-host/blob/main/charts/steadybit-extension-host/values.yaml) of the
extension.

#### Alternative (via own helm chart)

If you need more control, you can install the extension via its
dedicated [helm-chart](https://github.com/steadybit/extension-host/blob/main/charts/steadybit-extension-host).

```bash
helm repo add steadybit-extension-host https://steadybit.github.io/extension-host
helm repo update
helm upgrade steadybit-extension-host \
    --install \
    --wait \
    --timeout 5m0s \
    --create-namespace \
    --namespace steadybit-agent \
    steadybit-extension-host/steadybit-extension-host
```

### Linux Package

Please use
our [agent-linux.sh script](https://docs.steadybit.com/install-and-configure/install-agent/install-on-linux-hosts)
to install the extension on your Linux machine. The script will download the latest version of the extension and install
it using the package manager.

After installing, configure the extension by editing `/etc/steadybit/extension-host` and then restart the service.

## Extension registration

Make sure that the extension is registered with the agent. In most cases this is done automatically. Please refer to
the [documentation](https://docs.steadybit.com/install-and-configure/install-agent/extension-registration) for more
information about extension registration and how to verify.

## Security

We try to limit the access needed for the extension to the absolute minimum. So the extension itself can run as a
non-root user on a read-only root file-system and will, by default, if deployed using the provided helm chart.

In order to execute certain actions the extension needs extended capabilities, see details below.

### Resource and network attacks

Resource attacks start `stress-ng` or other resource attacking processes, and network attacks start `ip` or `tc` processes,
as runc container (sidecar) using the root user (`uid=0`, `gid=0`) and reusing the target container's linux namespace(s)
and control group(s). These processes are short-lived and terminated after the attack is finished.

This requires the following capabilities:
`CAP_SYS_ADMIN`, `CAP_SYS_CHROOT`, `CAP_SYS_RESOURCE`, `CAP_SYS_BOOT`, `CAP_NET_RAW`, `CAP_SYS_TIME`, `CAP_SYS_PTRACE`,
`CAP_KILL`, `CAP_NET_ADMIN`, `CAP_DAC_OVERRIDE`, `CAP_SETUID`, `CAP_SETGID`, `CAP_AUDIT_WRITE`

## Troubleshooting

Using cgroups v2 on the host and `nsdelegate` to mount the cgroup filesystem will prevent
the action from running processes in other cgroups (e.g. stress cpu/memory, disk fill).
In that case you need to remount the cgroup filesystem without the `nsdelegate` option.

```sh
sudo mount -o remount,rw,nosuid,nodev,noexec,relatime -t cgroup2 none /sys/fs/cgroup
```
