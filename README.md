<img src="./logo.svg" height="130" align="right" alt="Host logo">

# Steadybit extension-host

This [Steadybit](https://www.steadybit.com/) extension provides a host discovery and various actions for host targets.

Learn about the capabilities of this extension in our [Reliability Hub](https://hub.steadybit.com/extension/com.github.steadybit.extension_host).

## Configuration

| Environment Variable                  | Meaning                                                         | Required | Default |
|---------------------------------------|-----------------------------------------------------------------|----------|---------|
| `STEADYBIT_DISCOVERY_ENV_LIST`        | List of environment variables to be added to discovered targets | no       |         |

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

## Installation

### Using Docker

```sh
$ docker run \
  --rm \
  -p 8085 \
  --name steadybit-extension-host \
  ghcr.io/steadybit/extension-host:latest
```

### Using Helm in Kubernetes

```sh
$ helm repo add steadybit-extension-host https://steadybit.github.io/extension-host
$ helm repo update
$ helm upgrade steadybit-extension-host \
    --install \
    --wait \
    --timeout 5m0s \
    --create-namespace \
    --namespace steadybit-extension \
    steadybit-extension-host/steadybit-extension-host
```

## Register the extension

Make sure to register the extension at the steadybit platform. Please refer to
the [documentation](https://docs.steadybit.com/integrate-with-steadybit/extensions/extension-installation) for more information.
