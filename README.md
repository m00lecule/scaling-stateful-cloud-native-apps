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

```zsh
docker run -d --restart=always -p "5000:5000" --name "registry" registry:2

docker network connect "kind" "registry"
```

3. metallb

```zsh
kubectl get configmap kube-proxy -n kube-system -o yaml | \
sed -e "s/strictARP: false/strictARP: true/" | \
kubectl apply -f - -n kube-system
```

`kind/metallb/config` ip range
```
docker network inspect -f '{{.IPAM.Config}}' kind
```

4. sticky sessions

```zsh
curl -I --cookie "INGRESSCOOKIE=fa2127219c775b67d5347fc68b10f36b|ad539e4d8906dea703a59719eea04c4d;" -X GET localhost/notes/5
```