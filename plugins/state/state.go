package state

import (
	pb "github.com/snarlysodboxer/hambone/generated"
)

type Interface interface {
	AddInstance(*pb.Instance) (int32, error)
	AddSpecGroup(*pb.SpecGroup) (int32, error)
	ListInstances() (map[int32]string, error)
	ListSpecGroups() (map[int32]string, error)
}
