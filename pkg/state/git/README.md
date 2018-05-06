# git State Store adapter

### Notes on this adapter

_The remaining notes and this adapter, while working is for those who can tollerate the occasional difficulties that arrise from the git server being down and from concurrency. (This would usually mean manually inspecting and cleaning in the repo, or deleting the pod it's running in and starting from the latest commit.) This code could well be improved but has been back-burnered in favor of working on the etcd adapter._

##### TODO
* ensure repo is clean and pulled at init time (push if needed)
* cluster-wide lock on any repo changes
* if update fails, reset repo, release lock, return error to caller
* if a committed push fails, retry and eventually, release lock, return error to caller, (shutdown, delete self-pod)?
* add atomicity via OldInstance (see etcd adapter)
* periodically check that working tree is clean?


