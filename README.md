# hambone

## CRUD differently configured copies of the same Kubernetes specs using Kustomize and Git

# Consists of a gRPC server (with grpc-gateway JSON adapter)

# The App

* hambone aims to be as simple and dumb as possible, and expect you to do almost all the validation client-side, where you will also build in your custom domain logic. // TODO think about this



## Instances

* The API is as simple as possible and does no validation itself, however it uses `kubectl apply` which validates by default, and 
* It's a simple CRUD app that writes arbitrary files in a structured way, and tracks all changes in git.


# Roadmap

* Track logged in users so Git commits can be properly Authored
