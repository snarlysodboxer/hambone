package main

import (
	"flag"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"strings"
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
	response, err := client.Create(context.Background(), specGroup)
	if err != nil {
		return "", err
	}
	return response.GetName(), nil
}

func readSpecGroup(client pb.SpecGroupsClient, name string) (*pb.SpecGroup, error) {
	specGroup, err := client.Read(context.Background(), &pb.Name{name})
	if err != nil {
		return &pb.SpecGroup{}, err
	}
	return specGroup, nil
}

func listSpecGroups(client pb.SpecGroupsClient) (*pb.StringList, error) {
	response, err := client.List(context.Background(), &pb.Empty{})
	if err != nil {
		return &pb.StringList{}, err
	}
	return response, nil
}

func updateSpecGroup(client pb.SpecGroupsClient) (string, error) {
	spec := &pb.Spec{Name: "Deployment", Template: `new text template {{.Asdf}}`}
	specs := []*pb.Spec{}
	specs = append(specs, spec)
	specGroup := &pb.SpecGroup{}
	specGroup.Name = "my-product"
	specGroup.Specs = specs
	response, err := client.Update(context.Background(), specGroup)
	if err != nil {
		return "", err
	}
	return response.GetName(), nil
}

func deleteSpecGroup(client pb.SpecGroupsClient, name string) (string, error) {
	returnedName, err := client.Delete(context.Background(), &pb.Name{name})
	if err != nil {
		return "", err
	}
	return returnedName.GetName(), nil
}

func createInstance(client pb.InstancesClient) (string, error) {
	instance := &pb.Instance{Name: "my-client", SpecGroupName: "my-product", ValueSets: []*pb.ValueSet{&pb.ValueSet{"Deployment", `{"Name": "my-client"}`}}}
	response, err := client.Create(context.Background(), instance)
	if err != nil {
		return "", err
	}
	return response.GetName(), nil
}

func readInstance(client pb.InstancesClient, name string) (*pb.Instance, error) {
	response, err := client.Read(context.Background(), &pb.Name{name})
	if err != nil {
		return &pb.Instance{}, err
	}
	return response, nil
}

func listInstances(client pb.InstancesClient) (*pb.StringMap, error) {
	response, err := client.List(context.Background(), &pb.Empty{})
	if err != nil {
		return &pb.StringMap{}, err
	}
	return response, nil
}

func updateInstance(client pb.InstancesClient) (string, error) {
	instance := &pb.Instance{Name: "my-client", SpecGroupName: "my-product", ValueSets: []*pb.ValueSet{&pb.ValueSet{"Deployment", `{"Name": "new-my-client"}`}}}
	response, err := client.Update(context.Background(), instance)
	if err != nil {
		return "", err
	}
	return response.GetName(), nil
}

func deleteInstance(client pb.InstancesClient, name string) (string, error) {
	response, err := client.Delete(context.Background(), &pb.Name{name})
	if err != nil {
		return "", err
	}
	return response.GetName(), nil
}

func renderInstance(client pb.InstancesClient) ([]string, error) {
	instance := &pb.Instance{Name: "my-client", SpecGroupName: "my-product", ValueSets: []*pb.ValueSet{&pb.ValueSet{"Deployment", `{"Name": "new-my-client"}`}}}
	rendereds, err := client.Render(context.Background(), instance)
	if err != nil {
		return []string{}, err
	}
	return rendereds.GetList(), nil
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
	case "listSpecGroups":
		specGroups, err := listSpecGroups(specGroupsClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Read SpecGroups: '%v'\n", specGroups)
	case "updateSpecGroup":
		name, err := updateSpecGroup(specGroupsClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Updated SpecGroup '%s'\n", name)
	case "deleteSpecGroup":
		name, err := deleteSpecGroup(specGroupsClient, "my-product")
		if err != nil {
			panic(err)
		}
		fmt.Printf("Deleted SpecGroup: '%v'\n", name)
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
	case "listInstances":
		instances, err := listInstances(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Read Instances: '%v'\n", instances)
	case "updateInstance":
		name, err := updateInstance(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Updated Instance '%s'\n", name)
	case "deleteInstance":
		name, err := deleteInstance(instancesClient, "my-client")
		if err != nil {
			panic(err)
		}
		fmt.Printf("Deleted Instance: '%v'\n", name)
	case "renderInstance":
		rendereds, err := renderInstance(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Rendered Instance Specs: '%s'\n", strings.Join(rendereds, "\n\n---\n\n"))
	default:
		fmt.Println("Unrecognized action")
	}
}
