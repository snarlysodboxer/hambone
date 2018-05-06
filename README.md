# hambone

## An API server for CRUDing differently configured copies of the same Kubernetes specs using [Kustomize](https://github.com/kubernetes/kubectl/tree/58f555205b015986f2e487dc88a1481b6de3c5c4/cmd/kustomize) and Git

### Design

* `hambone` consists of a gRPC server (with grpc-gateway JSON adapter)
* `hambone` is designed to be run in replica in Kubernetes
* `hambone` and it's API aim to be as simple and dumb as possible, and expect you to do almost all the validation client-side where you can also build in your custom domain logic, or obtain external information for secrets or disk volume IDs, etc.
* `hambone` uses `kubectl apply` and `kustomize build` both of which validate YAML, and `kubectl` validates objects. Care is taken to return meaningful errors.
* `hambone` writes `kustomization.yaml` files in a structured way, and tracks all changes in Git. It rejects any `kustomization.yml` file changes which are rejected by Kubernetes.
* `hambone` writes arbitrary files at specified paths and tracks all changes in Git. It does not run any files.
* `hambone` expects you to create your own `kustomize` bases to use in your Instances, either manually in the repo, or through the `CustomFile` RPC call.

### Dependencies

_It's a purposeful design choice to execute shell commands rather than using the Kubernetes and Git APIs directly. This makes it easy to support most versions of Kubernetes/kubectl and Git, and helps to keep this app simple._

* We use the following external executables
    * `sh`
    * `test`
    * `kubectl`
    * `git` (Only for `git` State Store adapter)
* Typical usage would be to package these executables together in a Docker image along with your Git repository, and then mount in credentials for `kubectl` and `git` when running a container.

* For the `etcd` State Store adapter, etcd version 3 is required

### Build/Run

* Normal
    * `go build -o hambone main.go && ./hambone`
* Debug
    * `go build -tags debug -o hambone main.go && ./hambone`

### Roadmap

* Add metrics
* Document better
* Tests
* Consider an additional API for CRUDing base configurations
* Consider soft delete, or functionality to shutdown K8s pods without deleting (suspend?)
* Git plugin
    * Consider switching to [plumbing commands](http://schacon.github.io/git/git.html#_low_level_commands_plumbing)
    * Has been converted to an interface, needs refactor
    * Track logged in users so Git commits can be properly Authored

