# hambone

## An API server for CRUDing differently configured copies of the same Kubernetes specs using [Kustomize](https://github.com/kubernetes/kubectl/tree/58f555205b015986f2e487dc88a1481b6de3c5c4/cmd/kustomize)

## Purpose

You want to practice [Declarative Application Management](https://github.com/kubernetes/kubectl/blob/cc7be26dd0fe2c11b5ac43c4dc0771767e6264e5/cmd/kustomize/docs/glossary.md#declarative-application-management) using Kubernetes, and you need to CRUD multiple copies of the same application(s), configured differently. For example Wordpress sites, or copies of a single tenant monolith for different clients.

`kustomize` does the heavy lifting, having the concept of a base set of Kubernetes YAML objects which you "kustomize" using "overlay"s (what `hambone` calls "Instances",) allowing you to override any value in the base YAML. `kustomize` isn't however usable by customer service reps, managers, or clients. An API server + clients can solve those needs.

### Features

* Supports multiple State Store adapters, currently `etcd` (recommended) and `git`. Additional adapters can be written to fulfill the [Interface](https://github.com/snarlysodboxer/hambone/blob/master/pkg/state/state.go). (PRs welcome!)
* Follows the 
* Aims to be as simple as possible, and expects you to do almost all the validation client-side where you can also build in your custom domain logic, or obtain external information for secrets or disk volume IDs, etc.
* Uses `kubectl apply` and `kustomize build` both of which validate YAML, and `kubectl` validates objects. Care is taken to return meaningful errors.
* Manages `kustomization.yaml` files ([Instances](docs/glossary.md#instance)) in a structured way, and tracks all changes in the State Store. The API safely rejects any `kustomization.yml` files which are rejected by `kustomize`, `kubectl`, or Kubernetes.
* The API allows a client to store and retrieve default configurations which can be used when creating new Instances. This allows clients to be written in a way that hides YAML and complexity from the end-user, and so non-technicals can CRUD,  E.G. a customer support/manager facing SPA, or a server side `hambone` client so customers can request their own Instances. This part of the API is TODO.
* Ready to be run in replica in Kubernetes. See examples (TODO).

### Design

* Consists of a gRPC server (with grpc-gateway JSON adapter TODO)
* The server can be run either with existing `kustomize` base(s) on the filesystem (such as a git repo,) or base and overlay files can be created and managed in the State Store through the API's `CustomFile` call/endpoint (`CustomFile` is TODO).
* With the `etcd` adapter, the server is concurrency safe. The `git` adapter makes every attempt, but it's hard to protect against every circumstance. Problems could need manually fixed in (hopefully) rare circumstances.
* Rather than using the Kubernetes and Git APIs directly, `hambone` takes advantage of  `kustomize` and `kubectl`. This makes it easy to support most versions of Kubernetes/kubectl and Git, and keeps this app simple.

### Dependencies

* The following external executables are needed
    * `sh`
    * `kubectl`
    * `kustomize`
* For the State Store adapter (either or)
	* Git
	    * `git` (installed and configured to auth to your repo)
	    * `test`
	* etcd
	    * `etcd` version 3
* Directory
    * `hambone` expects to be executed from the same directory as you would run `kustomize`.
* Base(s)
    * You must create your own `kustomize` base(s) to use in your Instances, either manually on the filesystem, or through the `CustomFile` call/endpoint (`CustomFile` is TODO). Here are some [examples](https://github.com/snarlysodboxer/hambone/tree/master/examples).

### Usage

#### Run in Docker

See `hambone --help` for configuration flags.

* `docker run -it --rm --network host -v ${PWD}:/hambone snarlysodboxer/hambone:v1.7.3` - `v1.7.3` corresponds to the `kubectl`/Kubernetes version.

One could build an image `FROM snarlysodboxer/hambone:v1.7.3` and add a Git repository containing `kustomize` base(s), and then mount in credentials for `kubectl` and if needed `git`.

#### Get the repo and build it yourself
    go get github.com/snarlysodboxer/hambone
    cd $GOPATH/src/github.com/snarlysodboxer/hambone/examples
###### With Docker and docker-compose
    docker-compose up
###### Without Docker
   * Setup prerequisites
        * Install and setup git, or install and run etcd.
        * Install and setup [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
        * Install [kustomize](https://github.com/kubernetes/kubectl/tree/cc7be26dd0fe2c11b5ac43c4dc0771767e6264e5/cmd/kustomize)
   * Run server

    go build -o ./hambone ../main.go && ./hambone # OR
    go build -tags debug -o ./hambone ../main.go && ./hambone

### Develop

* TODO
* If you want to test against minikube, install and start it.
* If you change the proto file, run `./bin/build-stubs.sh` to regenerate the Protocol Buffer code.

### Roadmap

* Create example Kubernetes specs for running the app
* Create example Kustomize base specs
* Add metrics
* Document better
* Tests
* Consider soft delete, or functionality to shutdown K8s pods without deleting (suspend?)
* Track logged in users so changes can be Authored by someone
* Git adapter
    * Consider switching to [plumbing commands](http://schacon.github.io/git/git.html#_low_level_commands_plumbing)
    * Has been crudely converted from in-line to an adapter fulfulling an interface, needs refactored.
* etcd adapter
    * Consider creating the ability to write all files to the filesystem and then `git commit` them. (For the purpose of manually working with the files in the State Store.)

