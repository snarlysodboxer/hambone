package main

import (
	"flag"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	template = `this text template for {{.Name}}`
)

var (
	action        = flag.String("action", "createSpecGroup", "Which action to run")
	serverAddress = flag.String("server_address", "127.0.0.1:50051", "Where to reach the server")
)

func createSpecGroup(client pb.SpecGroupsClient) (int32, error) {
	spec := &pb.Spec{Name: "Deployment", Template: template}
	specs := []*pb.Spec{}
	specs = append(specs, spec)
	specGroup := &pb.SpecGroup{}
	specGroup.Name = "my-product"
	specGroup.Specs = specs
	request := &pb.CreateSpecGroupRequest{}
	request.SpecGroup = specGroup
	id, err := client.Create(context.Background(), request)
	if err != nil {
		return 0, err
	}
	return id.GetId(), nil
}

func readSpecGroup(client pb.SpecGroupsClient, id int32) (*pb.SpecGroup, error) {
	request := &pb.ReadSpecGroupRequest{id}
	specGroup, err := client.Read(context.Background(), request)
	if err != nil {
		return &pb.SpecGroup{}, err
	}
	return specGroup.SpecGroup, nil
}

func createInstance(client pb.InstancesClient) (int32, error) {
	instance := &pb.Instance{Name: "my-client", SpecGroupId: 1, ValueSets: []*pb.ValueSet{&pb.ValueSet{1, `{"Name": "my-client"}`}}}
	request := &pb.CreateInstanceRequest{instance}
	id, err := client.Create(context.Background(), request)
	if err != nil {
		return 0, err
	}
	return id.GetId(), nil
}

func readInstance(client pb.InstancesClient, id int32) (*pb.Instance, error) {
	request := &pb.ReadInstanceRequest{id}
	instance, err := client.Read(context.Background(), request)
	if err != nil {
		return &pb.Instance{}, err
	}
	return instance.Instance, nil
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
	case "readSpecGroup":
		specGroup, err := readSpecGroup(specGroupsClient, 1)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Read SpecGroup: %v\n", specGroup)
	case "createInstance":
		id, err := createInstance(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Println("Created Instance with ID: ", id)
	case "readInstance":
		instance, err := readInstance(instancesClient, 1)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Read Instance: %v\n", instance)
	default:
		fmt.Println("Unrecognized action")
	}
}
