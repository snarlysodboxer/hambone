package main

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"golang.org/x/net/context"
)

type instancesServer struct {
}

func newInstancesServer() *instancesServer {
	return &instancesServer{}
}

// Create adds the given Instance to the list, and applies it to the Kubernetes cluster
func (server *instancesServer) Create(ctx context.Context, createInstanceRequest *pb.CreateInstanceRequest) (*pb.CreateInstanceResponse, error) {
	response := &pb.CreateInstanceResponse{}
	instance := createInstanceRequest.Instance
	err := renderer.Render(instance)
	if err != nil {
		return response, err
	}
	id, err := stateStore.AddInstance(instance)
	if err != nil {
		return response, err
	}
	err = k8sEngine.ApplyInstance(instance)
	if err != nil {
		return response, err
	}
	response.Id = id
	return response, nil
}
