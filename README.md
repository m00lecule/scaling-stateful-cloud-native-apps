# scaling-stateful-cloud-native-applications

## requirements

- `go` 1.18
- `kind`
- `kustomize`
- `kubectl`
- `pre-commit`
- `docker-compose`
- `docker`
- `locust` 2.9.0

## setup docker-compose

```zsh
docker-compose up
```

to rebuild app image:

```zsh
 docker-compose up --build app
```

## setup local kubernetes cluster

1. kind

```zsh
kind create cluster --name scaling-stateful --config cluster.yaml

kubectl cluster-info --context kind-scaling-stateful

kubectl apply -f pv.yaml
```

2. local docker registry

docker registry is required to inject own images into kind - [ref](https://kind.sigs.k8s.io/docs/user/local-registry/)

```zsh
docker run -d --restart=always -p "5000:5000" --name "registry" registry:2

docker network connect "kind" "registry"
```

```zsh
docker build . -t stateful-app:1.0
docker tag stateful-app:1.0 localhost:5000/stateful-app:1.0
docker push localhost:5000/stateful-app:1.0
```

3. kind kustomize
3.1. metallb

```zsh
kustomize build kind/metallb | kubectl apply -f -
kubectl get configmap kube-proxy -n kube-system -o yaml | \
sed -e "s/strictARP: false/strictARP: true/" | \
kubectl apply -f - -n kube-system
```

`kind/metallb/config` ip range
```
docker network inspect -f '{{.IPAM.Config}}' kind
```

3.2. nginx ingress

```zsh
kustomize build kind/nginx-ingress | kubectl apply -f -
```

4. app kustomize

```zsh
kustomize build kustomize/app | kubectl apply -f -
```

5. sticky sessions

```zsh
curl -I --cookie "INGRESSCOOKIE=fa2127219c775b67d5347fc68b10f36b|ad539e4d8906dea703a59719eea04c4d;" -X GET localhost/notes/5

curl -vvv -X PATCH --cookie 'INGRESSCOOKIE=4fe5ccb0dfcccdaf4b986f7b884a65ed|ad539e4d8906dea703a59719eea04c4d;' localhost/carts/1 -d @../test.json
```

# performance tests

## Locust

```zsh
docker-compose up -d
locust -f locust/locustfile.py --headless --conf locust/locust.conf # or skip --headless and start tests from web ui - http://0.0.0.0:8089
```