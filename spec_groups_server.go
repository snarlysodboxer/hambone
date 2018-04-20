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
}

// Create adds the given SpecGroup to the list
func (server *specGroupsServer) Create(ctx context.Context, createSpecGroupRequest *pb.CreateSpecGroupRequest) (*pb.CreateSpecGroupResponse, error) {
	id, err := server.stateStore.CreateSpecGroup(createSpecGroupRequest.SpecGroup)
	if err != nil {
		return &pb.CreateSpecGroupResponse{}, err
	}
	return &pb.CreateSpecGroupResponse{Id: id}, nil
}

// Read returns the SpecGroup for the given id
func (server *specGroupsServer) Read(ctx context.Context, readSpecGroupRequest *pb.ReadSpecGroupRequest) (*pb.ReadSpecGroupResponse, error) {
	specGroup, err := server.stateStore.ReadSpecGroup(readSpecGroupRequest.Id)
	if err != nil {
		return &pb.ReadSpecGroupResponse{}, err
	}
	return &pb.ReadSpecGroupResponse{SpecGroup: specGroup}, nil
}
