package main

import (
	"flag"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/engine"
	"github.com/snarlysodboxer/hambone/plugins/render"
	"github.com/snarlysodboxer/hambone/plugins/state"
	"google.golang.org/grpc"
	"net"
	"plugin"
)

var (
	rendererPluginPath   = flag.String("render_plugin", "./plugins/render/default/default.so", "Path to a render plugin file")
	stateStorePluginPath = flag.String("state_plugin", "./plugins/state/etcd/etcd.so", "Path to a state store plugin file")
	k8sEnginePluginPath  = flag.String("engine_plugin", "./plugins/engine/k8s-api/k8s-api.so", "Path to an engine plugin file")
	listenAddress        = flag.String("listen_address", "127.0.0.1:50051", "The network address upon which the server should listen")
	renderer             render.Interface
	stateStore           state.Interface
	k8sEngine            engine.Interface
)

func setStateStorePlugin(filePath string) {
	stateStorePlugin, err := plugin.Open(filePath)
	if err != nil {
		panic(err)
	}
	stateStoreSymbol, err := stateStorePlugin.Lookup("StateStore")
	if err != nil {
		panic(err)
	}
	store, ok := stateStoreSymbol.(state.Interface)
	if !ok {
		panic("Unexpected type from StateStore plugin")
	}
	stateStore = store
}

func setRendererPlugin(filePath string) {
	renderPlugin, err := plugin.Open(filePath)
	if err != nil {
		panic(err)
	}
	rendererSymbol, err := renderPlugin.Lookup("Renderer")
	if err != nil {
		panic(err)
	}
	rend, ok := rendererSymbol.(render.Interface)
	if !ok {
		panic("Unexpected type from Renderer plugin")
	}
	renderer = rend
}

func setK8sEnginePlugin(filePath string) {
	k8sEnginePlugin, err := plugin.Open(filePath)
	if err != nil {
		panic(err)
	}
	k8sEngineSymbol, err := k8sEnginePlugin.Lookup("K8sEngine")
	if err != nil {
		panic(err)
	}
	eng, ok := k8sEngineSymbol.(engine.Interface)
	if !ok {
		panic("Unexpected type from K8sEngine plugin")
	}
	k8sEngine = eng
}

func main() {
	flag.Parse()

	setStateStorePlugin(*stateStorePluginPath)
	setRendererPlugin(*rendererPluginPath)
	setK8sEnginePlugin(*k8sEnginePluginPath)

	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		panic(err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	specGroupsServer := newSpecGroupsServer()
	pb.RegisterSpecGroupsServer(grpcServer, specGroupsServer)

	instancesServer := newInstancesServer()
	pb.RegisterInstancesServer(grpcServer, instancesServer)

	fmt.Printf("Listening on %s\n", *listenAddress)
	err = grpcServer.Serve(listener)
	if err != nil {
		panic(err)
	}
}
