package main

import (
	"flag"
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/instances"
	"github.com/snarlysodboxer/hambone/pkg/state"
	"github.com/snarlysodboxer/hambone/pkg/state/etcd"
	"github.com/snarlysodboxer/hambone/pkg/state/git"
	_ "google.golang.org/genproto/googleapis/api/annotations" // dummy import for dep
	"google.golang.org/grpc"
	"log"
	"net"
)

// TODO catch SIGTERM and shutdown gRPC server

var (
	listenAddress = flag.String("listen_address", "127.0.0.1:50051", "The network address upon which the server should listen")
	instancesDir  = flag.String("instances_dir", "./instances", "The directory in which to create instance directories")
	statePlugin   = flag.String("state_store", "etcd", "State store adapter to use, git or etcd")
	etcdEndpoints = flag.String("etcd_endpoints", "http://127.0.0.1:2379", "Comma-separated list of etcd endpoints, only used for etcd adapter")
)

func main() {
	flag.Parse()

	var stateStore state.StateEngine
	switch *statePlugin {
	case "git":
		stateStore = &git.GitEngine{}
	case "etcd":
		stateStore = &etcd.EtcdEngine{*etcdEndpoints}
	default:
		panic("Please choose `git` or `etcd` for state_store option")
	}
	stateStore.Init()

	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		panic(err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	instancesServer := instances.NewInstancesServer(*instancesDir, stateStore)
	pb.RegisterInstancesServer(grpcServer, instancesServer)

	log.Printf("Listening on %s\n", *listenAddress)
	err = grpcServer.Serve(listener)
	if err != nil {
		panic(err)
	}
}
