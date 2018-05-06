package main

import (
	"flag"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	kustomizationYamlThird = `namePrefix: my-third-client-

commonLabels:
  client: my-third-client
  myProductVersion: '3.1'

commonAnnotations:
  TAM: joel

secretGenerator:
- name: my-product-app-key
  commands:
    app-key: "echo $PWD"
  type: Opaque

bases:
- ../../my-product

patches:
- ../../versions/3.1.yml
`
	kustomizationYamlOther = `namePrefix: my-other-client-

commonLabels:
  client: my-other-client
  myProductVersion: '3.6'

commonAnnotations:
  TAM: joel

secretGenerator:
- name: my-product-app-key
  commands:
    app-key: "echo $PWD"
  type: Opaque

bases:
- ../../my-product

patches:
- ../../versions/3.6.yml
`
	kustomizationYaml = `namePrefix: my-client-

commonLabels:
  client: my-client
  myProductVersion: '3.1'

commonAnnotations:
  TAM: joel

secretGenerator:
- name: my-product-app-key
  commands:
    app-key: "echo $PWD"
  type: Opaque

bases:
- ../../my-product

patches:
- ../../versions/3.1.yml
`
	kustomizationYamlMod = `namePrefix: my-client-

commonLabels:
  client: my-client
  myProductVersion: '3.6'

commonAnnotations:
  TAM: joel

secretGenerator:
- name: my-product-app-key
  commands:
    app-key: "echo $PWD"
  type: Opaque

bases:
- ../../my-product

patches:
- ../../versions/3.6.yml
`
)

var (
	action        = flag.String("action", "applyInstance", "Which action to run")
	serverAddress = flag.String("server_address", "127.0.0.1:50051", "Where to reach the server")
)

func applyInstance(client pb.InstancesClient) (*pb.Instance, error) {
	instance := &pb.Instance{Name: "my-client", KustomizationYaml: kustomizationYaml}
	// instance := &pb.Instance{Name: "my-other-client", KustomizationYaml: kustomizationYamlOther}
	// instance := &pb.Instance{Name: "my-third-client", KustomizationYaml: kustomizationYamlThird}
	instance, err := client.Apply(context.Background(), instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

// atomic update
func updateInstance(client pb.InstancesClient) (*pb.Instance, error) {
	// instance := &pb.Instance{Name: "my-client", KustomizationYaml: kustomizationYamlMod}
	// instance.OldInstance = &pb.Instance{Name: "my-client", KustomizationYaml: kustomizationYaml}
	instance := &pb.Instance{Name: "my-client", KustomizationYaml: kustomizationYaml}
	instance.OldInstance = &pb.Instance{Name: "my-client", KustomizationYaml: kustomizationYamlMod}
	instance, err := client.Apply(context.Background(), instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

// If a name is passed in GetOptions, Start and Stop are ignored
func getInstance(client pb.InstancesClient) (*pb.InstanceList, error) {
	getOptions := &pb.GetOptions{Name: "my-client"}
	instanceList, err := client.Get(context.Background(), getOptions)
	if err != nil {
		return instanceList, err
	}
	return instanceList, nil
}

func getInstances(client pb.InstancesClient) (*pb.InstanceList, error) {
	// getOptions := &pb.GetOptions{Start: 1, Stop: 10, ExcludeStatuses: true} // Get paginated and/or without statuses
	getOptions := &pb.GetOptions{} // Get all, including statuses
	instanceList, err := client.Get(context.Background(), getOptions)
	if err != nil {
		return instanceList, err
	}
	return instanceList, nil
}

func deleteInstance(client pb.InstancesClient) (*pb.Instance, error) {
	// instance := &pb.Instance{Name: "my-client"}
	instance := &pb.Instance{Name: "my-other-client"}
	// instance := &pb.Instance{Name: "my-third-client"}
	instance, err := client.Delete(context.Background(), instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

func atomicDeleteInstance(client pb.InstancesClient) (*pb.Instance, error) {
	clientName := "my-client"
	// clientName := "my-other-client"
	// clientName := "my-third-client"
	instance := &pb.Instance{Name: clientName}
	instance.OldInstance = &pb.Instance{Name: clientName, KustomizationYaml: kustomizationYaml}
	instance, err := client.Delete(context.Background(), instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

func main() {
	flag.Parse()

	opts := []grpc.DialOption{grpc.WithInsecure()}
	connection, err := grpc.Dial(*serverAddress, opts...)
	if err != nil {
		panic(err)
	}
	defer connection.Close()
	instancesClient := pb.NewInstancesClient(connection)

	switch *action {
	case "applyInstance":
		instance, err := applyInstance(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Applied Instance '%v'\n", instance)
	case "updateInstance":
		instance, err := updateInstance(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Updated Instance '%v'\n", instance)
	case "getInstance":
		instanceList, err := getInstance(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Got InstanceList '%v'\n", instanceList)
	case "getInstances":
		instanceList, err := getInstances(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Got InstanceList '%v'\n", instanceList)
	case "deleteInstance":
		instance, err := deleteInstance(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Deleted Instance '%v'\n", instance)
	case "atomicDeleteInstance":
		instance, err := atomicDeleteInstance(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Atomically Deleted Instance '%v'\n", instance)
	default:
		fmt.Println("Unrecognized action")
	}
}
