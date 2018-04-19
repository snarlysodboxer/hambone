package main

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/engine"
	"github.com/snarlysodboxer/hambone/plugins/render"
	"github.com/snarlysodboxer/hambone/plugins/state"
	"golang.org/x/net/context"
	"plugin"
)

type instancesServer struct {
	stateStore state.Interface
	k8sEngine  engine.Interface
	renderer   render.Interface
}

func newInstancesServer() *instancesServer {
	return &instancesServer{}
}

func (server *instancesServer) setStateStorePlugin(filePath string) {
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
		panic("Unexpected type from StateStore plugin")
	}
	server.stateStore = stateStore
}

func (server *instancesServer) setRendererPlugin(filePath string) {
	renderPlugin, err := plugin.Open(filePath)
	if err != nil {
		panic(err)
	}
	rendererSymbol, err := renderPlugin.Lookup("Renderer")
	if err != nil {
		panic(err)
	}
	var renderer render.Interface
	renderer, ok := rendererSymbol.(render.Interface)
	if !ok {
		panic("Unexpected type from Renderer plugin")
	}
	server.renderer = renderer
}

func (server *instancesServer) setK8sEnginePlugin(filePath string) {
	k8sEnginePlugin, err := plugin.Open(filePath)
	if err != nil {
		panic(err)
	}
	k8sEngineSymbol, err := k8sEnginePlugin.Lookup("K8sEngine")
	if err != nil {
		panic(err)
	}
	var k8sEngine engine.Interface
	k8sEngine, ok := k8sEngineSymbol.(engine.Interface)
	if !ok {
		panic("Unexpected type from K8sEngine plugin")
	}
	server.k8sEngine = k8sEngine
}

// Create adds the given Instance to the list, and applies it to the Kubernetes cluster
func (server *instancesServer) Create(ctx context.Context, createInstanceRequest *pb.CreateInstanceRequest) (*pb.CreateInstanceResponse, error) {
	response := &pb.CreateInstanceResponse{}
	instance := createInstanceRequest.Instance
	err := server.renderer.Render(instance)
	if err != nil {
		return response, err
	}
	id, err := server.stateStore.AddInstance(instance)
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
