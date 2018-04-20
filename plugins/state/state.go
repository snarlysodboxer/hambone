package state

import (
	pb "github.com/snarlysodboxer/hambone/generated"
)

type Interface interface {
	NextInstanceID() (int32, error)
	CreateInstance(*pb.Instance) (int32, error)
	ListInstances() (map[int32]string, error)

	NextSpecGroupID() (int32, error)
	CreateSpecGroup(*pb.SpecGroup) (int32, error)
	ReadSpecGroup(int32) (*pb.SpecGroup, error)
	ListSpecGroups() (map[int32]string, error)
}
