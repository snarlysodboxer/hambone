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
func (server *specGroupsServer) Create(ctx context.Context, createSpecGroupRequest *pb.CreateSpecGroupRequest) (*pb.CreateSpecGroupResponse, error) {
	name, err := server.stateStore.CreateSpecGroup(createSpecGroupRequest.SpecGroup)
	if err != nil {
		return &pb.CreateSpecGroupResponse{}, err
	}
	return &pb.CreateSpecGroupResponse{Name: name}, nil
}

// Read returns the SpecGroup for the given name
func (server *specGroupsServer) Read(ctx context.Context, readSpecGroupRequest *pb.ReadSpecGroupRequest) (*pb.ReadSpecGroupResponse, error) {
	specGroup, err := server.stateStore.ReadSpecGroup(readSpecGroupRequest.Name)
	if err != nil {
		return &pb.ReadSpecGroupResponse{}, err
	}
	return &pb.ReadSpecGroupResponse{SpecGroup: specGroup}, nil
}
