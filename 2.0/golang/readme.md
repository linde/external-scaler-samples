
How i got things working and verified:

```bash

export KIND_CLUSTER_NAME=keda-custom2
export KO_DOCKER_REPO=kind.local 


kind create cluster --name=${KIND_CLUSTER_NAME}
kubectl apply --server-side -f https://github.com/kedacore/keda/releases/download/v2.14.1/keda-2.14.1-core.yaml

# fyi uses KIND_CLUSTER_NAME and KO_DOCKER_REPO
ko build --base-import-paths   

kubectl apply -f deploy.yaml
```

To see what's working:

```bash

kubectl port-forward service/golang-external-scaler  -n golang-external-scaler-ns 6000 &

grpcurl   -plaintext -import-path .  -proto externalscaler.proto  \
    []:6000 externalscaler.ExternalScaler.GetMetrics

# using the default metric size
grpcurl   -plaintext -import-path .  -proto externalscaler.proto   \
    []:6000 externalscaler.ExternalScaler.IsActive

# watch over the course of 5 minutes it scale out and back down
watch -d kubectl get -n golang-external-scaler-ns  pod,deployment,scaledobject 

## FYI: if running locally via `go run main.go`, you can pass it in explicitly.
# the value for a cluster's ScaledObject will trump this if hitting the cluster service
grpcurl  -d '{"scalerMetadata":{"metricTargetSize":"1", "metricModulus":"3"}}' \
    -plaintext -import-path .  -proto externalscaler.proto   \
    []:6000 externalscaler.ExternalScaler.IsActive


```