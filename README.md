# Sidecar Injector

The sidecar injector can be used to inject a sidecar into a Pod via a [Mutating Webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).

## Usage

The sidecar injector can be installed via Helm. To use the Helm [cert-manager](https://cert-manager.io) is required.

```sh
helm repo add ricoberger https://ricoberger.github.io/helm-charts
helm install sidecar-injector ricoberger/sidecar-injector
```

The configuration for the injected sidecars can be passed to the sidecar injector via the `config` value in the Helm chart. The following configuration injects the basic auth sidecar:

```yaml
config: |
  containers:
    - name: basic-auth
      image: ricoberger/sidecar-injector:basic-auth
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

## Test

To test the sidecar injector locally you can run the `./test/test.sh` shell script. This script will create a kind cluster and installs Istio and the cert-manager into the cluster. After that it will install the sidecar injector and a service named echoserver.

In the test we inject the basic auth sidecar into the echoserver and use the [external authorization](https://istio.io/latest/docs/tasks/security/authorization/authz-custom/) feature of Istio to protect the service, when it is accessed via the `echoserver.127.0.0.1.nip.io` host.

When the `test.sh` script was executed, you can use the following cURL command to test the setup:

```sh
curl -vvv -u admin:admin http://echoserver.127.0.0.1.nip.io
```
