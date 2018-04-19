package main

import (
	"flag"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"google.golang.org/grpc"
	"net"
)

var (
	rendererPluginPath   = flag.String("render_plugin", "./plugins/render/default/default.so", "Path to a render plugin file")
	stateStorePluginPath = flag.String("state_plugin", "./plugins/state/etcd/etcd.so", "Path to a state store plugin file")
	k8sEnginePluginPath  = flag.String("engine_plugin", "./plugins/engine/k8s-api/k8s-api.so", "Path to an engine plugin file")
	listenAddress        = flag.String("listen_address", "127.0.0.1:50051", "The network address upon which the server should listen")
)

func main() {
	flag.Parse()

	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		panic(err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	specGroupsServer := newSpecGroupsServer()
	specGroupsServer.setStateStorePlugin(*stateStorePluginPath)
	pb.RegisterSpecGroupsServer(grpcServer, specGroupsServer)

	instancesServer := newInstancesServer()
	instancesServer.setStateStorePlugin(*stateStorePluginPath)
	instancesServer.setRendererPlugin(*rendererPluginPath)
	instancesServer.setK8sEnginePlugin(*k8sEnginePluginPath)
	pb.RegisterInstancesServer(grpcServer, instancesServer)

	fmt.Printf("Listening on %s\n", *listenAddress)
	err = grpcServer.Serve(listener)
	if err != nil {
		panic(err)
	}
}
