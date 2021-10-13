# scaling-stateful-cloud-native-applications

## requirements

- `go` 1.17
- `kind`
- `kustomize`
- `kubectl`
- `pre-commit`

## boostrap

1. kubernetes cluster

```zsh
kind create cluster --name scaling-stateful --config cluster.yaml

kubectl cluster-info --context kind-scaling-stateful

kubectl apply -f pv.yaml
```

2. docker registry

docker registry is required to inject own images into kind - [ref](https://kind.sigs.k8s.io/docs/user/local-registry/)