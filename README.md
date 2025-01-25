# Sidecar Injector

The sidecar injector can be used to inject a sidecar into a Pod via a [Mutating Webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).

## Usage

The sidecar injector can be installed via Helm. To use the Helm [cert-manager](https://cert-manager.io) is required.

```sh
helm upgrade --install sidecar-injector oci://ghcr.io/ricoberger/charts/sidecar-injector --version 1.0.0
```

The configuration for the injected sidecars can be passed to the sidecar injector via the `config` value in the Helm chart. The following configuration injects the basic auth sidecar:

```yaml
config: |
  containers:
    - name: basic-auth
      image: ghrc.io/ricoberger/sidecar-injector:basic-auth
      imagePullPolicy: Always
      env:
        - name: BASIC_AUTH_PASSWORD
          valueFrom:
            secretKeyRef:
              key: BASIC_AUTH_PASSWORD
              name: basic-auth-credentials
        - name: BASIC_AUTH_USERNAME
          valueFrom:
            secretKeyRef:
              key: BASIC_AUTH_USERNAME
              name: basic-auth-credentials
      ports:
        - name: http-auth
          containerPort: 4180
      livenessProbe:
        httpGet:
          port: 4180
          path: /health
        initialDelaySeconds: 1
        timeoutSeconds: 5
      readinessProbe:
        httpGet:
          port: 4180
          path: /health
        initialDelaySeconds: 1
        timeoutSeconds: 5
      resources:
        requests:
          cpu: 50m
          memory: 64Mi
        limits:
          cpu: 50m
          memory: 64Mi
  volumes: []
  environmentVariables: []
```

You can also define a list of volumes and a list of environment variables, which should be set from Pod annotations.

When the sidecar injector is installed in your cluster you have to set some annotation for your Pods:

- `sidecar-injector.ricoberger.de: enabled`: Enable the sidecar injection for a Pod.
- `sidecar-injector.ricoberger.de/containers: <CONTAINER-NAME-1>,<CONTAINER-NAME-2>`: Comma-separated list of container names, which should be used from the configuration file.
- `sidecar-injector.ricoberger.de/init-containers: <CONTAINER-NAME-1>,<CONTAINER-NAME-2>`: Comma-separated list of container names, which should be used from the configuration file as init containers.
- `sidecar-injector.ricoberger.de/volumes: <VOLUME-NAME-1>,<VOLUME-NAME-2>`: Comma-separated list of volume names, which should be used from the configuration file.

### Environment Variables

It is possible to set additional environment variables for the injected sidecar via annotations. The environment variables which can be injected must be defined in the `environmentVariables` section in the config, e.g.

```yaml
config: |
  environmentVariables:
    - name: ENV_NAME
      container: <CONTAINER-NAME>
      annotation: sidecar-injector.ricoberger.de/envname
```

With this configuration a user can then use the `sidecar-injector.ricoberger.de/envname` annotation to set the value of the `ENV_NAME` environment variable in the specified `<CONTAINER-NAME>`:

```yaml
---
kind: Deployment
apiVersion: apps/v1
metadata:
  name: example
  namespace: default
spec:
  selector:
    matchLabels:
      app: example
  template:
    metadata:
      annotations:
        sidecar-injector.ricoberger.de: enabled
        sidecar-injector.ricoberger.de/envname: envvalue
```

### Resources

Since the injected sidecars might need different resources depending on the service where they are injected it is also possible to overwrite the CPU Requests / Limits and Memory Requests and Limits via an annotation:

- `sidecar-injector.ricoberger.de/containers/<CONTAINER-NAME>/cpurequests`
- `sidecar-injector.ricoberger.de/containers/<CONTAINER-NAME>/cpulimits`
- `sidecar-injector.ricoberger.de/containers/<CONTAINER-NAME>/memoryrequests`
- `sidecar-injector.ricoberger.de/containers/<CONTAINER-NAME>/memorylimits`

The same can be done for init containers by using the following annotations:

- `sidecar-injector.ricoberger.de/init-containers/<CONTAINER-NAME>/cpurequests`
- `sidecar-injector.ricoberger.de/init-containers/<CONTAINER-NAME>/cpulimits`
- `sidecar-injector.ricoberger.de/init-containers/<CONTAINER-NAME>/memoryrequests`
- `sidecar-injector.ricoberger.de/init-containers/<CONTAINER-NAME>/memorylimits`
