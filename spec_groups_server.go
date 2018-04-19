package main

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/state"
	"golang.org/x/net/context"
	"plugin"
)

type specGroupsServer struct {
	stateStore state.Interface
}

func newSpecGroupsServer() *specGroupsServer {
	return &specGroupsServer{}
}

func (server *specGroupsServer) setStateStorePlugin(filePath string) {
	stateStorePlugin, err := plugin.Open(filePath)
	if err != nil {
		panic(err)
	}
	stateStoreSymbol, err := stateStorePlugin.Lookup("StateStore")
	if err != nil {
		panic(err)
	}
	var stateStore state.Interface
	stateStore, ok := stateStoreSymbol.(state.Interface)
	if !ok {
		panic("Unexpected type from state store plugin")
	}
	server.stateStore = stateStore
}

// Create adds the given SpecGroup to the list
func (server *specGroupsServer) Create(ctx context.Context, specGroupRequest *pb.CreateSpecGroupRequest) (*pb.CreateSpecGroupResponse, error) {
	id, err := server.stateStore.AddSpecGroup(specGroupRequest.SpecGroup)
	if err != nil {
		return &pb.CreateSpecGroupResponse{}, err
	}
	return &pb.CreateSpecGroupResponse{Id: id}, nil
}
