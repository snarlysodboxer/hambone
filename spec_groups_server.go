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
func (server *specGroupsServer) Create(ctx context.Context, specGroupRequest *pb.CreateSpecGroupRequest) (*pb.CreateSpecGroupResponse, error) {
	id, err := server.stateStore.CreateSpecGroup(specGroupRequest.SpecGroup)
	if err != nil {
		return &pb.CreateSpecGroupResponse{}, err
	}
	return &pb.CreateSpecGroupResponse{Id: id}, nil
}
