# How to run E2E tests in local
## Setup kind

Install kind, please refer: https://kind.sigs.k8s.io/

And bootstrap a cluster.

```
$ K8S_VERSION=1.29.4 ./hack/kind-with-registry.sh
```

Install argo-rollout.

```
$ kubectl create namespace argo-rollouts
$ kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml
```

## Build docker and push
```
$ export IMAGE_ID=localhost:5000/rollout-optimizer-controller
$ docker build . --tag ${IMAGE_ID}:e2e
$ docker push ${IMAGE_ID}:e2e
```

## Execute e2e test
```
$ go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo
$ export MANAGER_IMAGE=${IMAGE_ID}:e2e
$ ginkgo -r ./e2e
```
