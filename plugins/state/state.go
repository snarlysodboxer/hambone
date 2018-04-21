package state

import (
	pb "github.com/snarlysodboxer/hambone/generated"
)

type Interface interface {
	Init()

	CreateInstance(*pb.Instance) (string, error)
	ReadInstance(string) (*pb.Instance, error)
	ListInstances() (map[string]string, error)

	CreateSpecGroup(*pb.SpecGroup) (string, error)
	ReadSpecGroup(string) (*pb.SpecGroup, error)
	ListSpecGroups() ([]string, error)
}
