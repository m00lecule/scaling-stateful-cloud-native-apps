# scaling-stateful-cloud-native-applications

## boostrap

1. kubernetes cluster

```zsh
kind create cluster --name scaling-stateful --config cluster.yaml

kubectl cluster-info --context kind-scaling-stateful

kubectl apply -f pv.yaml
```