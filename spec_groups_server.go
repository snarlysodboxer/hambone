package main

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/state"
	"golang.org/x/net/context"
)

type specGroupsServer struct {
	stateStore state.Interface
}

func newSpecGroupsServer() *specGroupsServer {
	return &specGroupsServer{}
}

func (server *specGroupsServer) SetStateStore(stateStore state.Interface) {
	server.stateStore = stateStore
	server.stateStore.Init()
}

// Create adds the given SpecGroup to the list
func (server *specGroupsServer) Create(ctx context.Context, specGroup *pb.SpecGroup) (*pb.Name, error) {
	name, err := server.stateStore.CreateSpecGroup(specGroup)
	if err != nil {
		return &pb.Name{}, err
	}
	return &pb.Name{Name: name}, nil
}

// Read returns the SpecGroup for the given name
func (server *specGroupsServer) Read(ctx context.Context, name *pb.Name) (*pb.SpecGroup, error) {
	specGroup, err := server.stateStore.ReadSpecGroup(name.Name)
	if err != nil {
		return &pb.SpecGroup{}, err
	}
	return specGroup, nil
}

// List returns a list of the Instances' Names
func (server *specGroupsServer) List(ctx context.Context, _ *pb.Empty) (*pb.StringList, error) {
	list, err := server.stateStore.ListSpecGroups()
	if err != nil {
		return &pb.StringList{}, err
	}
	return &pb.StringList{list}, nil
}
