package main

import (
	"flag"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	kustomizationYamlOther = `namePrefix: my-other-client-

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
)

var (
	action        = flag.String("action", "applyInstance", "Which action to run")
	serverAddress = flag.String("server_address", "127.0.0.1:50051", "Where to reach the server")
)

func applyInstance(client pb.InstancesClient) (*pb.Instance, error) {
	// instance := &pb.Instance{Name: "my-client", KustomizationYaml: kustomizationYaml}
	instance := &pb.Instance{Name: "my-other-client", KustomizationYaml: kustomizationYamlOther}
	instance, err := client.Apply(context.Background(), instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

func getInstance(client pb.InstancesClient) (*pb.InstanceList, error) {
	getOptions := &pb.GetOptions{Name: "my-client"}
	instanceList, err := client.Get(context.Background(), getOptions)
	if err != nil {
		return instanceList, err
	}
	return instanceList, nil
}

func getInstances(client pb.InstancesClient) (*pb.InstanceList, error) {
	getOptions := &pb.GetOptions{Start: 1, Stop: 1, ExcludeStatuses: true}
	instanceList, err := client.Get(context.Background(), getOptions)
	if err != nil {
		return instanceList, err
	}
	return instanceList, nil
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
	default:
		fmt.Println("Unrecognized action")
	}
}
