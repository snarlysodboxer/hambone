package main

import (
	"flag"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	template = `this text template from {{.Place}}`
)

var (
	action        = flag.String("action", "createSpecGroup", "Which action to run")
	serverAddress = flag.String("server_address", "127.0.0.1:50051", "Where to reach the server")
)

func createSpecGroup(client pb.SpecGroupsClient) (int32, error) {
	spec := &pb.Spec{123, "my-product", template}
	specs := []*pb.Spec{}
	specs = append(specs, spec)
	specGroup := &pb.SpecGroup{}
	specGroup.Specs = specs
	request := &pb.CreateSpecGroupRequest{}
	request.SpecGroup = specGroup
	id, err := client.Create(context.Background(), request)
	if err != nil {
		return 0, err
	}
	return id.GetId(), nil
}

func createInstance(client pb.InstancesClient) (int32, error) {
	instance := &pb.Instance{456, "my-client", 123, []*pb.ValueSet{&pb.ValueSet{1, `{"Name": "my-client"}`}}}
	request := &pb.CreateInstanceRequest{instance}
	id, err := client.Create(context.Background(), request)
	if err != nil {
		return 0, err
	}
	return id.GetId(), nil
}

func main() {
	flag.Parse()

	opts := []grpc.DialOption{grpc.WithInsecure()}
	connection, err := grpc.Dial(*serverAddress, opts...)
	if err != nil {
		panic(err)
	}
	defer connection.Close()
	specGroupsClient := pb.NewSpecGroupsClient(connection)
	instancesClient := pb.NewInstancesClient(connection)

	switch *action {
	case "createSpecGroup":
		id, err := createSpecGroup(specGroupsClient)
		if err != nil {
			panic(err)
		}
		fmt.Println("Created SpecGroup with ID: ", id)
	case "createInstance":
		id, err := createInstance(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Println("Created Instance with ID: ", id)
	default:
		fmt.Println("Unrecognized action")
	}
}
