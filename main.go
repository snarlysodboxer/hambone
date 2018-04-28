package main

import (
	"flag"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/instances"
	"google.golang.org/grpc"
	"net"
)

var (
	listenAddress = flag.String("listen_address", "127.0.0.1:50051", "The network address upon which the server should listen")
	instancesDir  = flag.String("instances_dir", "./instances", "The root directory in which to create instance directories")
)

func main() {
	flag.Parse()

	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		panic(err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	instancesServer := instances.NewInstancesServer(*instancesDir)
	pb.RegisterInstancesServer(grpcServer, instancesServer)

	fmt.Printf("Listening on %s\n", *listenAddress)
	err = grpcServer.Serve(listener)
	if err != nil {
		panic(err)
	}
}
