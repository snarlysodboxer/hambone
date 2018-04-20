package main

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/engine"
	"github.com/snarlysodboxer/hambone/plugins/render"
	"github.com/snarlysodboxer/hambone/plugins/state"
	"golang.org/x/net/context"
)

type instancesServer struct {
	renderer   render.Interface
	stateStore state.Interface
	k8sEngine  engine.Interface
}

func newInstancesServer() *instancesServer {
	return &instancesServer{}
}

func (server *instancesServer) SetRenderer(renderer render.Interface) {
	server.renderer = renderer
}

func (server *instancesServer) SetEngine(engine engine.Interface) {
	server.k8sEngine = engine
}

func (server *instancesServer) SetStateStore(stateStore state.Interface) {
	server.stateStore = stateStore
}

// Create adds the given Instance to the list, and applies it to the Kubernetes cluster
func (server *instancesServer) Create(ctx context.Context, createInstanceRequest *pb.CreateInstanceRequest) (*pb.CreateInstanceResponse, error) {
	response := &pb.CreateInstanceResponse{}
	instance := createInstanceRequest.Instance
	id, err := server.stateStore.CreateInstance(instance)
	if err != nil {
		return response, err
	}
	err = server.k8sEngine.ApplyInstance(instance)
	if err != nil {
		return response, err
	}
	response.Id = id
	return response, nil
}

// Reads reads and returns an Instance for the given id
func (server *instancesServer) Read(ctx context.Context, readInstanceRequest *pb.ReadInstanceRequest) (*pb.ReadInstanceResponse, error) {
	response := &pb.ReadInstanceResponse{}
	instance, err := server.stateStore.ReadInstance(readInstanceRequest.Id)
	if err != nil {
		return response, err
	}
	response.Instance = instance
	return response, nil
}
