# git State Store adapter

### Notes on this adapter

_Significant effort has been made to make this adapter fault tolerant, but this is hard. This adapter is for those who can tolerate the occasional difficulties that arrise from the git server being down or other network hiccups. (This might mean deleting the pod hambone is running in and starting from the latest commit.)_

This adapter does support etcd locks so that it can be run in replica.

##### TODO
* if a committed push fails, retry and eventually, release lock, return error to caller, (shutdown, self-delete pod)?

