package main

import (
	"flag"
	"log"
	"net"

	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/instances"
	"github.com/snarlysodboxer/hambone/pkg/state"
	"github.com/snarlysodboxer/hambone/pkg/state/etcd"
	"github.com/snarlysodboxer/hambone/pkg/state/git"
	_ "google.golang.org/genproto/googleapis/api/annotations" // dummy import for dep
	"google.golang.org/grpc"
)

// TODO catch SIGTERM and shutdown gRPC server

var (
	listenAddress        = flag.String("listen_address", "127.0.0.1:50051", "The network address upon which the server should listen")
	repoDir              = flag.String("repo_dir", ".", "The Git repository directory")
	instancesDir         = flag.String("instances_dir", "./instances", "The directory in which to create instance directories (relative to repo_dir)")
	templatesDir         = flag.String("templates_dir", "./templates", "The directory in which instance templates are stored (relative to repo_dir)")
	statePlugin          = flag.String("state_store", "etcd", "State store adapter to use, git or etcd")
	etcdEndpoints        = flag.String("etcd_endpoints", "http://127.0.0.1:2379", "Comma-separated list of etcd endpoints, only used for etcd adapter")
	enableKustomizeBuild = flag.Bool("enable_kustomize_build", false, "Enable Kustomize build to verify builds work")
	enableKubectl        = flag.Bool("enable_kubectl", false, "Enable Kustomize build and kubectl apply and delete, managing objects in the local cluster. Implies enable_kustomize_build")
)

func main() {
	flag.Parse()

	var stateStore state.Engine
	switch *statePlugin {
	case "git":
		stateStore = &git.Engine{WorkingDir: *repoDir}
	case "etcd":
		stateStore = &etcd.Engine{EndpointsString: *etcdEndpoints}
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

	instancesServer := instances.NewInstancesServer(*instancesDir, *templatesDir, stateStore, *enableKustomizeBuild, *enableKubectl)
	pb.RegisterInstancesServer(grpcServer, instancesServer)

	log.Printf("Listening on %s\n", *listenAddress)
	err = grpcServer.Serve(listener)
	if err != nil {
		panic(err)
	}
}
