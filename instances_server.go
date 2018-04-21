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
	server.stateStore.Init()
}

// Create adds the given Instance to the list, and applies it to the Kubernetes cluster
func (server *instancesServer) Create(ctx context.Context, instance *pb.Instance) (*pb.Name, error) {
	name, err := server.stateStore.CreateInstance(instance)
	if err != nil {
		return &pb.Name{}, err
	}
	err = server.k8sEngine.ApplyInstance(instance)
	if err != nil {
		return &pb.Name{}, err
	}
	return &pb.Name{name}, nil
}

// Reads and returns an Instance for the given name
func (server *instancesServer) Read(ctx context.Context, name *pb.Name) (*pb.Instance, error) {
	instance, err := server.stateStore.ReadInstance(name.Name)
	if err != nil {
		return &pb.Instance{}, err
	}
	return instance, nil
}

// List returns a list of Instance Names mapped to their respective SpecGroup Names
func (server *instancesServer) List(ctx context.Context, _ *pb.Empty) (*pb.StringMap, error) {
	instances, err := server.stateStore.ListInstances()
	if err != nil {
		return &pb.StringMap{}, err
	}
	return &pb.StringMap{instances}, nil
}
