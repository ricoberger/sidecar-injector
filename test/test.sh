#!/bin/sh
set -o errexit

# create registry container unless it already exists
reg_name='kind-registry'
reg_port='5000'
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  docker run \
    -d --restart=always -p "127.0.0.1:${reg_port}:5000" --name "${reg_name}" \
    registry:2
fi

# create a cluster with the local registry enabled in containerd and some extra port mappings to use the Istio Ingress
# Gateway as ingress controller.
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: test
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_name}:${reg_port}"]
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 30000
    hostPort: 80
    listenAddress: "127.0.0.1"
    protocol: TCP
  - containerPort: 30001
    hostPort: 443
    listenAddress: "127.0.0.1"
    protocol: TCP
  - containerPort: 30002
    hostPort: 15021
    listenAddress: "127.0.0.1"
    protocol: TCP
EOF

# connect the registry to the cluster network
# (the network may already be connected)
docker network connect "kind" "${reg_name}" || true

# Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

# Build the image and push the image to our local registry
docker build . -f cmd/webhook/Dockerfile -t localhost:5000/sidecar-injector:test
docker push localhost:5000/sidecar-injector:test

docker build . -f cmd/basicauth/Dockerfile -t localhost:5000/sidecar-injector:basic-auth
docker push localhost:5000/sidecar-injector:basic-auth

# Deploy the Istio Operator, Istio and the cert-manager as requirements for the sidecar-injector.
kubectl apply -f ./test/istio-operator.yaml
sleep 10
kubectl wait pod --namespace=istio-operator -l name=istio-operator --for=condition=Ready --timeout=180s

kubectl apply -f ./test/istio.yaml
sleep 10
kubectl wait pod --namespace=istio-system -l app=istiod --for=condition=Ready --timeout=180s
kubectl apply -f ./test/istioresources.yaml

kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.3.1/cert-manager.yaml
sleep 10
kubectl wait pod --namespace=cert-manager -l app=cert-manager --for=condition=Ready --timeout=180s
kubectl wait pod --namespace=cert-manager -l app=webhook --for=condition=Ready --timeout=180s
sleep 30

# Deploy the sidecar-injector
helm upgrade --install sidecar-injector ./charts/sidecar-injector --namespace=istio-system -f ./test/sidecar-injector.yaml
sleep 10
kubectl wait pod --namespace=istio-system -l app.kubernetes.io/name=sidecar-injector --for=condition=Ready --timeout=180s

# Deploy the echoserver to test the sidecar-injector
kubectl apply -f ./test/echoserver.yaml
