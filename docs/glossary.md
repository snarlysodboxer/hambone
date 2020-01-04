# Instance

A particular instantiation of a `kustomize` base. `kustomize` calls this an `overlay`.

# State Store

Storage for Instances and other arbitrary files. Currently `git` or `etcd`.

# Overlay

An overlay `kustomization.yaml` equals a `hambone` `Instance`

# Template

A special `Instance` for clients to use when creating new `Instance`s. For example you could store Go template files.

