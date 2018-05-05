package state

import (
	pb "github.com/snarlysodboxer/hambone/generated"
)

type Updater interface {
	Init() error
	Cancel(error) error
	Commit() error
}

type Deleter interface {
	Init() error
	Cancel(error) error
	Commit() error
}

type Getter interface {
	Run() error
}

type StateEngine interface {
	Init() error
	NewUpdater(*pb.Instance, string) Updater
	NewDeleter(*pb.Instance, string) Deleter
	NewGetter(*pb.GetOptions, *pb.InstanceList, string) Getter
}
