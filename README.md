# hambone

# NOTE: taking a new direction with [kustomize](https://github.com/kubernetes/kubectl/tree/master/cmd/kustomize)

## CRUD differently configured copies of the same Kubernetes specs

### Consists of a gRPC server (with grpc-gateway JSON adapter) and clients

### Design Considerations

* `CustomResourceDefinition`s in lieu of storing state externally was considered. 

### Roadmap

* Consider using [Kustomize](https://github.com/kubernetes/kubectl/tree/master/cmd/kustomize) instead of Go templates

// TODO setup dep
* Handle errors in `main.go` better
* Setup vendoring with `dep`
