# Steadybit extension-host

This extension provides a Host discovery and the following attacks for host targets:

 - stress CPU
 - stress Memory
 - stress Disk
 -

## Configuration

| Environment Variable                  | Meaning                                                         | Default |
|---------------------------------------|-----------------------------------------------------------------|---------|
| `STEADYBIT_DISCOVERY_ENV_LIST`        | List of environment variables to be added to discovered targets |         |

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

## Running the Extension

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
