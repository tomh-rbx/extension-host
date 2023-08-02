<img src="./logo.svg" height="130" align="right" alt="Host logo">

# Steadybit extension-host

This [Steadybit](https://www.steadybit.com/) extension provides a host discovery and various actions for host targets.

Learn about the capabilities of this extension in our [Reliability Hub](https://hub.steadybit.com/extension/com.steadybit.extension_host).

## Configuration

| Environment Variable            | Meaning                                                                                                                                                                                                                       | Required | Default |
|---------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `STEADYBIT_LABEL_<key>=<value>` | Environment variables starting with `STEADYBIT_LABEL_` will be added to discovered targets' attributes. <br>**Example:** `STEADYBIT_LABEL_TEAM=Fullfillment` adds to each discovered target the attribute `team=Fullfillment` | no       |         |
| `STEADYBIT_DISCOVERY_ENV_LIST`  | List of environment variables to be evaluated and added to discovered targets' attributes. <br> **Example:** `STEADYBIT_DISCOVERY_ENV_LIST=STAGE` adds to each target the attribute `stage=<value of $STAGE>`                 | no       |         |

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

## Needed capabilities

The capabilities needed by this extension are: (which are provided by the helm chart)

- SYS_ADMIN
- SYS_RESOURCE
- SYS_BOOT
- NET_RAW
- SYS_TIME
- SYS_PTRACE
- KILL
- NET_ADMIN
- DAC_OVERRIDE
- SETUID
- SETGID
- AUDIT_WRITE

## Installation

### Using Helm in Kubernetes

```sh
helm repo add steadybit-extension-host https://steadybit.github.io/extension-host
helm repo update
helm upgrade steadybit-extension-host \
    --install \
    --wait \
    --timeout 5m0s \
    --create-namespace \
    --namespace steadybit-extension \
    steadybit-extension-host/steadybit-extension-host
```

### Using Docker

This extension is by default deployed using
our [outpost.sh docker compose script](https://docs.steadybit.com/install-and-configure/install-outpost-agent-preview/install-as-docker-container).

Or you can run it manually:

```sh
docker run \
  --rm \
  -p 8085:8085 \
  --name steadybit-extension-host \
  --privileged
  --network=host
  --pid=host
  ghcr.io/steadybit/extension-host:latest
```

### Linux Package

Please use our [outpost-linux.sh script](https://docs.steadybit.com/install-and-configure/install-outpost-agent-preview/install-on-linux-hosts) to install the
extension on your Linux machine.
The script will download the latest version of the extension and install it using the package manager.

## Register the extension

Make sure to register the extension at the steadybit platform. Please refer to
the [documentation](https://docs.steadybit.com/integrate-with-steadybit/extensions/extension-installation) for more information.
