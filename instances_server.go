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

// Create adds the given Instance to the StateStore, and applies it to the Kubernetes cluster
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

// Reads returns an Instance from the StateStore for the given name
func (server *instancesServer) Read(ctx context.Context, name *pb.Name) (*pb.Instance, error) {
	instance, err := server.stateStore.ReadInstance(name.Name)
	if err != nil {
		return &pb.Instance{}, err
	}
	return instance, nil
}

// List returns a list of Instance Names from the StateStore mapped to their respective SpecGroup Names
func (server *instancesServer) List(ctx context.Context, _ *pb.Empty) (*pb.StringMap, error) {
	instances, err := server.stateStore.ListInstances()
	if err != nil {
		return &pb.StringMap{}, err
	}
	return &pb.StringMap{instances}, nil
}

// Update updates the given Instance in the StateStore, and applies it to the Kubernetes cluster
func (server *instancesServer) Update(ctx context.Context, instance *pb.Instance) (*pb.Name, error) {
	name, err := server.stateStore.UpdateInstance(instance)
	if err != nil {
		return &pb.Name{}, err
	}
	err = server.k8sEngine.ApplyInstance(instance)
	if err != nil {
		return &pb.Name{}, err
	}
	return &pb.Name{name}, nil
}

// Delete deletes an Instance from the Kubernetes cluster and the StateStore for the given name
func (server *instancesServer) Delete(ctx context.Context, name *pb.Name) (*pb.Name, error) {
	instance, err := server.stateStore.ReadInstance(name.Name)
	if err != nil {
		return &pb.Name{}, err
	}
	err = server.k8sEngine.DeleteInstance(instance)
	if err != nil {
		return &pb.Name{}, err
	}
	returnedName, err := server.stateStore.DeleteInstance(name.Name)
	if err != nil {
		return &pb.Name{}, err
	}
	return &pb.Name{returnedName}, nil
}

// Render renders and returns each ValueSet in an Instance
func (server *instancesServer) Render(ctx context.Context, instance *pb.Instance) (*pb.StringList, error) {
	rendereds, err := server.renderer.Render(instance)
	if err != nil {
		return &pb.StringList{}, err
	}
	return &pb.StringList{rendereds}, nil
}

// Status returns status from Kubernetes for each ValueSet in an Instance
// func (server *instancesServer) Status(ctx context.Context, name *pb.Name) (*pb.StatusMessage, error) {
//     status, err := server.k8sEngine.Status(instance)
//     if err != nil {
//         return &pb.StatusMessage{}, err
//     }
//     return status, nil
// }
