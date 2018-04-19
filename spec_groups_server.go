package main

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"golang.org/x/net/context"
)

type specGroupsServer struct {
}

func newSpecGroupsServer() *specGroupsServer {
	return &specGroupsServer{}
}

// Create adds the given SpecGroup to the list
func (server *specGroupsServer) Create(ctx context.Context, specGroupRequest *pb.CreateSpecGroupRequest) (*pb.CreateSpecGroupResponse, error) {
	id, err := stateStore.AddSpecGroup(specGroupRequest.SpecGroup)
	if err != nil {
		return &pb.CreateSpecGroupResponse{}, err
	}
	return &pb.CreateSpecGroupResponse{Id: id}, nil
}
