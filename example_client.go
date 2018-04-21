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

func createSpecGroup(client pb.SpecGroupsClient) (string, error) {
	spec := &pb.Spec{Name: "Deployment", Template: template}
	specs := []*pb.Spec{}
	specs = append(specs, spec)
	specGroup := &pb.SpecGroup{}
	specGroup.Name = "my-product"
	specGroup.Specs = specs
	request := &pb.CreateSpecGroupRequest{}
	request.SpecGroup = specGroup
	response, err := client.Create(context.Background(), request)
	if err != nil {
		return "", err
	}
	return response.GetName(), nil
}

func readSpecGroup(client pb.SpecGroupsClient, name string) (*pb.SpecGroup, error) {
	request := &pb.ReadSpecGroupRequest{name}
	specGroup, err := client.Read(context.Background(), request)
	if err != nil {
		return &pb.SpecGroup{}, err
	}
	return specGroup.SpecGroup, nil
}

func createInstance(client pb.InstancesClient) (string, error) {
	instance := &pb.Instance{Name: "my-client", SpecGroupName: "my-product", ValueSets: []*pb.ValueSet{&pb.ValueSet{"Deployment", `{"Name": "my-client"}`}}}
	request := &pb.CreateInstanceRequest{instance}
	response, err := client.Create(context.Background(), request)
	if err != nil {
		return "", err
	}
	return response.GetName(), nil
}

func readInstance(client pb.InstancesClient, name string) (*pb.Instance, error) {
	request := &pb.ReadInstanceRequest{name}
	response, err := client.Read(context.Background(), request)
	if err != nil {
		return &pb.Instance{}, err
	}
	return response.Instance, nil
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
		name, err := createSpecGroup(specGroupsClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Created SpecGroup '%s'\n", name)
	case "readSpecGroup":
		specGroup, err := readSpecGroup(specGroupsClient, "my-product")
		if err != nil {
			panic(err)
		}
		fmt.Printf("Read SpecGroup: '%v'\n", specGroup)
	case "createInstance":
		name, err := createInstance(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Created Instance '%s'\n", name)
	case "readInstance":
		instance, err := readInstance(instancesClient, "my-client")
		if err != nil {
			panic(err)
		}
		fmt.Printf("Read Instance: '%v'\n", instance)
	default:
		fmt.Println("Unrecognized action")
	}
}
