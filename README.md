# hambone

## An API server for CRUDing differently configured copies of the same Kubernetes specs using [Kustomize](https://kustomize.io/)

## Purpose

You want to practice [Declarative Application Management](https://kubernetes.io/docs/concepts/overview/working-with-objects/object-management/#declarative-object-configuration) using Kubernetes, and you need to CRUD multiple copies of the same application(s), configured differently. For example Wordpress sites, or copies of a single tenant monolith for different clients.

Kustomize does the heavy lifting, having the concept of a Base set of Kubernetes YAML files which you customize using Overlays (which Hambone calls Instances,) allowing you to override any value in the base YAML. Hambone is an API server for automating Kustomize Overlays.

### Features

* Supports multiple State Store adapters, currently `etcd` (recommended) and `git`. Additional adapters can be written to fulfill the [Interface](https://github.com/snarlysodboxer/hambone/blob/master/pkg/state/state.go). (PRs welcome!)
* Aims to be as simple as possible, and expects you to do almost all the validation client-side where you can also build in your custom domain logic, or obtain external information for secrets or disk volume IDs, etc.
* Optionally uses `kustomize build` and `kubectl apply`, both of which validate YAML, and `kubectl` validates objects. Care is taken to return meaningful errors.
* Manages `kustomization.yaml` files ([Instances](docs/glossary.md#instance)) in a structured way, and tracks all changes in the State Store. When using the `kustomize` and `kubectl` options, the server safely rejects any configs which are rejected by those tools or by Kubernetes.
* Clients can pass an old version of an Instance when updating or deleting to prevent writes from stale reads.
* The API enables a client to store and retrieve Template configurations which can be used when creating new Instances.
* The `etcd` adapter uses [Distributed Locks](https://coreos.com/etcd/docs/latest/dev-guide/api_concurrency_reference_v3.html) for concurrency control.
* Ready to be run in replica in Kubernetes. See examples (TODO).

### Design

* Consists of a gRPC server (with grpc-gateway JSON adapter TODO)
* The server can be run with existing `kustomize` base(s) on the filesystem (such as a git repo,) or base and overlay files can be created and managed in the State Store through the API's `CustomFile` call/endpoint (`CustomFile` is TODO).
* With the `etcd` adapter, the server is concurrency safe. The `git` adapter makes every attempt, but it's hard to protect against every circumstance. Problems could need manually fixed in (hopefully) rare circumstances.
* Rather than using the Kubernetes and Git APIs directly, `hambone` takes advantage of the `kustomize` and `kubectl` binaries. This makes it easy to support most versions of Kubernetes/kubectl and Git, and keeps the app simple.
* It's the client's responsibility to ensure generated objects don't colide with other generated objects if they're going to be applied to the same cluster. This is a matter of configuring your Kustomize files correctly.

### Dependencies

* The following external executables are needed
    * `kubectl`
    * `kustomize`
* For the State Store adapter (either or)
    * Git
        * `sh`
        * `git` (installed and configured to auth to your repo)
        * `test`
    * etcd
        * `etcd` version 3
* Directory
    * `hambone` expects to be executed from the same directory as you would run `kustomize`.
* Base(s)
    * You must create your own `kustomize` base(s) to use in your Instances, either manually on the filesystem, or through the `CustomFile` call/endpoint (`CustomFile` is TODO). See the working [example base](https://github.com/snarlysodboxer/hambone/tree/master/examples).

### Usage

See `hambone --help` for configuration flags.

#### Run a Docker container

* `docker run -it --rm --network host -v ${PWD}:/hambone snarlysodboxer/hambone:v1.7.3` - The `v1.7.3` tag corresponds to the `kubectl`/Kubernetes version. See [Docker repo](https://hub.docker.com/r/snarlysodboxer/hambone/) for other tags.

One could build an image `FROM snarlysodboxer/hambone:<tag>` and add a Git repository containing `kustomize` base(s), and then mount in credentials for `kubectl` and if needed `git`.

#### Run the example

    go get github.com/snarlysodboxer/hambone

    cd $GOPATH/src/github.com/snarlysodboxer/hambone

##### With Docker and docker-compose

* Run the server

      cd examples && docker-compose up

* Run the example client

      ./bin/go.sh run examples/client.go -h

##### Without Docker

* Setup prerequisites
   * Install and setup git, or install and run etcd.
   * Install and setup [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
   * Install [kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md)
* Run the server

      cd examples && go build -o ./hambone ../main.go && ./hambone

* Or with debug logging

      cd examples && go build -tags debug -o ./hambone ../main.go && ./hambone

* Run the example client

      go run examples/client.go -h

### Develop

* Ensure your `~/.kube/config` is pointed to the cluster you want to develop against. I like to use minikube.
* If you change the proto file, regenerate the Protocol Buffer code
      ./bin/build-stubs.sh
* Run the tests

### Running the tests

* The unit tests can be run with
  * `go test ./...`
* There are a number of integration tests that rely on a running git server.
* There are a number of integration tests that rely on a running etcd server.
  * Both of these servers can be run with
  * `docker-compose up`
* The integration tests can be run with
  * `go test -tags=integration pkg/state/git/git_integration_test.go pkg/state/git/git.go`
  * `go test -tags=integration main_integration_test.go`
  * `go test -tags=integration ./...` - This can cause race conditions because of testing concurrency.
  * Debug mode can be turned on with
  * `go test -tags=integration,debug pkg/state/git/git_integration_test.go pkg/state/git/git.go`

### Roadmap

* Create example specs for running `hambone` in Kubernetes
* Add metrics
* Document better
* More tests
* Track logged in users so changes can be Authored by someone
* Allow passing arguments to Kustomize
* Return rich gRPC status Errors
* Git adapter
    * Add ability to use etcd for distributed locking.
    * Consider switching to [plumbing commands](http://schacon.github.io/git/git.html#_low_level_commands_plumbing)
    * Has been crudely converted from in-line to an adapter fulfulling an interface, needs refactored.
* etcd adapter
    * Consider creating the ability to write all files to the filesystem and then `git commit` them. (For the purpose of manually working with the files in the State Store.)

