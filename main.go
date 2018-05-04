package main

import (
	"flag"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/instances"
	"github.com/snarlysodboxer/hambone/plugins/state"
	"google.golang.org/grpc"
	"net"
	"plugin"
)

var (
	listenAddress   = flag.String("listen_address", "127.0.0.1:50051", "The network address upon which the server should listen")
	instancesDir    = flag.String("instances_dir", "./instances", "The root directory in which to create instance directories")
	statePluginPath = flag.String("state_plugin", "./git.so", "Path to a state store plugin file")
)

func getStatePlugin(filePath string) state.StateEngine {
	stateStorePlugin, err := plugin.Open(filePath)
	if err != nil {
		panic(err)
	}
	stateStoreSymbol, err := stateStorePlugin.Lookup("StateStore")
	if err != nil {
		panic(err)
	}
	store, ok := stateStoreSymbol.(state.StateEngine)
	if !ok {
		panic("Unexpected type from StateStore plugin")
	}
	return store
}

func main() {
	flag.Parse()

	stateStore := getStatePlugin(*statePluginPath)
	stateStore.Init()

	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		panic(err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	instancesServer := instances.NewInstancesServer(*instancesDir, stateStore)
	pb.RegisterInstancesServer(grpcServer, instancesServer)

	fmt.Printf("Listening on %s\n", *listenAddress)
	err = grpcServer.Serve(listener)
	if err != nil {
		panic(err)
	}
}
