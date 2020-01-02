package main

import (
	"flag"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	action        = flag.String("action", "applyInstance", "Which action to run")
	name          = flag.String("name", "my-client-1", "The name of the Instance to act upon")
	serverAddress = flag.String("server_address", "127.0.0.1:50051", "Where to reach the server")
)

func applyManyInstances(client pb.InstancesClient) error {
	for i := 0; i < 700; i++ {
		name := fmt.Sprintf("my-client-%d", i)
		kustomizationYaml := generateKustomizationYaml(name, name)
		instance := &pb.Instance{Name: name, KustomizationYaml: kustomizationYaml}
		instance, err := client.Apply(context.Background(), instance)
		if err != nil {
			return err
		}
	}
	return nil
}

func applyInstance(client pb.InstancesClient) (*pb.Instance, error) {
	kustomizationYaml := generateKustomizationYaml(*name, *name)
	instance := &pb.Instance{Name: *name, KustomizationYaml: kustomizationYaml}
	instance, err := client.Apply(context.Background(), instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

// atomic update
func updateInstance(client pb.InstancesClient) (*pb.Instance, error) {
	kustomizationYaml := generateKustomizationYaml(*name, *name)
	kustomizationYamlMod := generateKustomizationYaml(*name, "other-name")
	instance := &pb.Instance{Name: *name, KustomizationYaml: kustomizationYaml}
	instance.OldInstance = &pb.Instance{Name: *name, KustomizationYaml: kustomizationYamlMod}
	instance, err := client.Apply(context.Background(), instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

// If a name is passed in GetOptions, Start and Stop are ignored
func getInstance(client pb.InstancesClient) (*pb.InstanceList, error) {
	getOptions := &pb.GetOptions{Name: *name}
	instanceList, err := client.Get(context.Background(), getOptions)
	if err != nil {
		return instanceList, err
	}
	return instanceList, nil
}

func getInstances(client pb.InstancesClient) (*pb.InstanceList, error) {
	// getOptions := &pb.GetOptions{Start: 1, Stop: 10, ExcludeStatuses: true} // Get paginated and/or without statuses
	getOptions := &pb.GetOptions{ExcludeStatuses: true} // statuses are also ignored if kubectl is not enabled
	// getOptions := &pb.GetOptions{} // Get all, including statuses
	instanceList, err := client.Get(context.Background(), getOptions)
	if err != nil {
		return instanceList, err
	}
	return instanceList, nil
}

func deleteInstance(client pb.InstancesClient) (*pb.Instance, error) {
	instance := &pb.Instance{Name: *name}
	instance, err := client.Delete(context.Background(), instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

func atomicDeleteInstance(client pb.InstancesClient) (*pb.Instance, error) {
	instance := &pb.Instance{Name: *name}
	kustomizationYaml := generateKustomizationYaml(*name, *name)
	instance.OldInstance = &pb.Instance{Name: *name, KustomizationYaml: kustomizationYaml}
	instance, err := client.Delete(context.Background(), instance)
	if err != nil {
		return instance, err
	}
	return instance, nil
}

func generateKustomizationYaml(name, dnsName string) string {
	return fmt.Sprintf(`---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namePrefix: %s-

resources:
- ../../base

configMapGenerator:
- name: my-configmap
  namespace: default
  behavior: create
  literals:
  - APP_URL=https://%s.example.com
  - SOME=thing

`, name, dnsName)
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
	case "applyManyInstances":
		err := applyManyInstances(instancesClient)
		if err != nil {
			panic(err)
		}
		fmt.Println("Applied many Instances")
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
