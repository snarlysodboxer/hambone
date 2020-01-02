// +build integration

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/instances"
	"github.com/snarlysodboxer/hambone/pkg/state"
	"github.com/snarlysodboxer/hambone/pkg/state/etcd"
	"github.com/snarlysodboxer/hambone/pkg/state/git"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	gitRepoAddress = flag.String("git_repo_address", "http://localhost:5000/hambone/test-hambone.git", "The Git clone address for testing against")
	name           = "my-client-123"
	testCase       = testCaseClient{}
	dialTimeout    = 10 * time.Second
)

type testCaseClient struct {
	name       string
	connection *grpc.ClientConn
	client     pb.InstancesClient
	dir        string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func setupForGit() (*grpc.ClientConn, pb.InstancesClient, string, string) {
	// setup git
	tempDir, err := ioutil.TempDir("", "hambone-git")
	if err != nil {
		panic(err)
	}
	err = os.Chdir(tempDir)
	if err != nil {
		panic(err)
	}
	args := []string{`clone`, *gitRepoAddress}
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%s: %s", err.Error(), string(output))
		panic(err)
	}

	// setup state store
	tempRepoDir := filepath.Join(tempDir, "test-hambone")
	stateStore := &git.Engine{WorkingDir: tempRepoDir, EndpointsString: *etcdEndpoints, EtcdLocksGitKey: *etcdLocksGitKey}
	err = stateStore.Init()
	if err != nil {
		panic(err)
	}

	// setup gRPC server
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	instancesServer := instances.NewInstancesServer("overlays", "templates", stateStore, false, false)
	pb.RegisterInstancesServer(grpcServer, instancesServer)
	serverAddress := "127.0.0.1"
	listener, err := net.Listen("tcp", serverAddress+":0") // next available port
	if err != nil {
		panic(err)
	}
	go func() {
		err = grpcServer.Serve(listener)
		if err != nil {
			panic(err)
		}
	}()

	// setup gRPC client
	dialOpts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithTimeout(dialTimeout)}
	listenAddress := fmt.Sprintf("%s:%d", serverAddress, listener.Addr().(*net.TCPAddr).Port)
	connection, err := grpc.Dial(listenAddress, dialOpts...)
	if err != nil {
		panic(err)
	}

	return connection, pb.NewInstancesClient(connection), tempDir, "git"
}

func setupForEtcd() (*grpc.ClientConn, pb.InstancesClient, string, string) {
	tempDir, err := ioutil.TempDir("", "hambone-etcd")
	if err != nil {
		panic(err)
	}
	tempRepoDir := filepath.Join(tempDir, "test-hambone")
	err = os.MkdirAll(tempRepoDir, 0755)
	if err != nil {
		panic(err)
	}

	// setup state store
	stateStore := &etcd.Engine{WorkingDir: tempRepoDir, EndpointsString: *etcdEndpoints}
	err = stateStore.Init()
	if err != nil {
		panic(err)
	}

	// setup gRPC server
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	instancesServer := instances.NewInstancesServer("overlays", "templates", stateStore, false, false)
	pb.RegisterInstancesServer(grpcServer, instancesServer)
	serverAddress := "127.0.0.1"
	listener, err := net.Listen("tcp", serverAddress+":0") // next available port
	if err != nil {
		panic(err)
	}
	go func() {
		err = grpcServer.Serve(listener)
		if err != nil {
			panic(err)
		}
	}()

	// setup gRPC client
	dialOpts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithTimeout(dialTimeout)}
	listenAddress := fmt.Sprintf("%s:%d", serverAddress, listener.Addr().(*net.TCPAddr).Port)
	connection, err := grpc.Dial(listenAddress, dialOpts...)
	if err != nil {
		panic(err)
	}

	return connection, pb.NewInstancesClient(connection), tempDir, "etcd"
}

func TestMain(m *testing.M) {
	flag.Parse()
	exitCodes := []int{}
	cleanupFuncs := []func(){}

	for _, fn := range []func() (*grpc.ClientConn, pb.InstancesClient, string, string){setupForGit, setupForEtcd} {
		connection, instancesClient, dir, name := fn()
		cleanupFuncs = append(cleanupFuncs, func() {
			if err := os.RemoveAll(dir); err != nil {
				fmt.Errorf(err.Error())
			}
			if err := connection.Close(); err != nil {
				fmt.Errorf(err.Error())
			}
		})
		testCase.name = name
		testCase.connection = connection
		testCase.client = instancesClient
		testCase.dir = dir
		exitCodes = append(exitCodes, m.Run())
	}
	for _, fn := range cleanupFuncs {
		fn()
	}
	exitCode := 0
	for _, code := range exitCodes {
		if code != 0 {
			exitCode = code
		}
	}
	os.Exit(exitCode)
}

func TestApplyInstance(t *testing.T) {
	t.Run(testCase.name, func(t *testing.T) {
		randomName := generateRandomName(name)
		kustomizationYaml := generateKustomizationYaml(randomName, randomName)
		file := &pb.File{Name: "my-app/deployment.yaml", Directory: "my-app", Contents: string(deploymentYaml)}
		instance := &pb.Instance{Name: randomName, KustomizationYaml: kustomizationYaml, Files: []*pb.File{file}}
		returnedInstance, err := testCase.client.Apply(context.Background(), instance)
		if err != nil {
			t.Fatal(err)
		}
		sentVSReturned(t, returnedInstance, instance)

		// clean up
		_, err = testCase.client.Delete(context.Background(), instance)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestUpdateInstance(t *testing.T) {
	t.Run(testCase.name, func(t *testing.T) {
		// setup
		kustomizationYaml := generateKustomizationYaml(name, "asdf")
		kustomizationYamlOld := generateKustomizationYaml(name, name)
		oldInstance := &pb.Instance{Name: name, KustomizationYaml: kustomizationYamlOld}
		_, err := testCase.client.Apply(context.Background(), oldInstance)
		if err != nil {
			t.Fatal(err)
		}

		// atomic update
		instance := &pb.Instance{Name: name, KustomizationYaml: kustomizationYaml}
		instance.OldInstance = oldInstance
		returnedInstance, err := testCase.client.Apply(context.Background(), instance)
		if err != nil {
			t.Fatal(err)
		}
		sentVSReturned(t, returnedInstance, instance)

		//   atomic update OldInstance with different name fails
		instance.OldInstance.Name = "my-client-234"
		_, err = testCase.client.Apply(context.Background(), instance)
		if err != nil {
			if status.Convert(err).Message() != status.Convert(instances.InstanceNameMismatchError).Message() {
				t.Errorf(`Expected "%s" error, Got: %s`, instances.InstanceNameMismatchError.Error(), err.Error())
			}
		} else {
			t.Error("Expected error using OldInstance with different name")
		}

		//   atomic update OldInstance KustomizationYaml differing in State Store fails
		instance.KustomizationYaml = generateKustomizationYaml(name, generateRandomName(name))
		instance.OldInstance.Name = name
		instance.OldInstance.KustomizationYaml = generateKustomizationYaml(name, generateRandomName(name))
		_, err = testCase.client.Apply(context.Background(), instance)
		if err != nil {
			if status.Convert(err).Message() != status.Convert(state.OldInstanceDiffersError).Message() {
				t.Errorf(`Expected "%s" error, Got: %s`, state.OldInstanceDiffersError.Error(), err.Error())
			}
		} else {
			t.Error("Expected error using OldInstance with differing kustomization.yaml")
		}

		//   atomic update OldInstance passed when there's no existing Instance fails
		newName := generateRandomName(name)
		newInstance := &pb.Instance{Name: newName, KustomizationYaml: ""}
		newInstance.OldInstance = &pb.Instance{Name: newName, KustomizationYaml: ""}
		_, err = testCase.client.Apply(context.Background(), newInstance)
		if err != nil {
			if status.Convert(err).Message() != status.Convert(state.InstanceNoExistError).Message() {
				t.Errorf(`Expected "%s" error, Got: %s`, state.InstanceNoExistError.Error(), err.Error())
			}
		} else {
			t.Error("Expected error passing OldInstance where no Instance exists")
		}

		// clean up
		instance.OldInstance = nil
		_, err = testCase.client.Delete(context.Background(), instance)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestDeleteInstance(t *testing.T) {
	t.Run(testCase.name, func(t *testing.T) {
		// regular delete
		randomName := generateRandomName(name)
		kustomizationYaml := generateKustomizationYaml(randomName, randomName)
		instance := &pb.Instance{Name: randomName, KustomizationYaml: kustomizationYaml}
		_, err := testCase.client.Apply(context.Background(), instance)
		if err != nil {
			t.Fatal(err)
		}
		returnedInstance, err := testCase.client.Delete(context.Background(), instance)
		if err != nil {
			t.Fatal(err)
		}
		sentVSReturned(t, returnedInstance, instance)

		// atomic delete setup
		randomName = generateRandomName(name)
		kustomizationYaml = generateKustomizationYaml(randomName, randomName)
		instance = &pb.Instance{Name: randomName, KustomizationYaml: kustomizationYaml}
		_, err = testCase.client.Apply(context.Background(), instance)
		if err != nil {
			t.Fatal(err)
		}

		//   atomic delete OldInstance with different name fails
		instance.OldInstance = &pb.Instance{Name: "my-client-234", KustomizationYaml: kustomizationYaml}
		_, err = testCase.client.Delete(context.Background(), instance)
		if err != nil {
			if status.Convert(err).Message() != status.Convert(instances.InstanceNameMismatchError).Message() {
				t.Fatalf(`Expected "%s" error, Got: %s`, instances.InstanceNameMismatchError.Error(), err.Error())
			}
		} else {
			t.Fatal("Expected error using OldInstance with different name")
		}

		//   atomic delete OldInstance KustomizationYaml differing in State Store fails
		instance.OldInstance = &pb.Instance{Name: randomName, KustomizationYaml: ""}
		_, err = testCase.client.Delete(context.Background(), instance)
		if err != nil {
			if status.Convert(err).Message() != status.Convert(state.OldInstanceDiffersError).Message() {
				t.Fatalf(`Expected "%s" error, Got: %s`, state.OldInstanceDiffersError.Error(), err.Error())
			}
		} else {
			t.Fatal("Expected error using OldInstance with differing kustomization.yaml")
		}

		//   atomic delete proper
		instance.OldInstance = &pb.Instance{Name: randomName, KustomizationYaml: kustomizationYaml}
		returnedInstance, err = testCase.client.Delete(context.Background(), instance)
		if err != nil {
			t.Fatal(err)
		}
		sentVSReturned(t, returnedInstance, instance)
	})
}

func TestGetInstance(t *testing.T) {
	t.Run(testCase.name, func(t *testing.T) {
		// create some instances
		kustomizationYaml := generateKustomizationYaml(name, name)
		instance1 := &pb.Instance{Name: name, KustomizationYaml: kustomizationYaml}
		_, err := testCase.client.Apply(context.Background(), instance1)
		if err != nil {
			t.Fatal(err)
		}
		name = "my-client-345"
		kustomizationYaml = generateKustomizationYaml(name, name)
		instance3 := &pb.Instance{Name: name, KustomizationYaml: kustomizationYaml}
		_, err = testCase.client.Apply(context.Background(), instance3)
		if err != nil {
			t.Fatal(err)
		}
		name = "my-client-456"
		kustomizationYaml = generateKustomizationYaml(name, name)
		instance4 := &pb.Instance{Name: name, KustomizationYaml: kustomizationYaml}
		_, err = testCase.client.Apply(context.Background(), instance4)
		if err != nil {
			t.Fatal(err)
		}

		// Get all, including statuses if kubectl is enabled
		getOptions := &pb.GetOptions{}
		instanceList, err := testCase.client.Get(context.Background(), getOptions)
		if err != nil {
			t.Fatal(err)
		}
		for _, received := range instanceList.Instances {
			for _, sent := range []*pb.Instance{instance1, instance3, instance4} {
				if sent.Name == received.Name {
					sentVSReturned(t, received, sent)
				}
			}
		}

		// Get paginated and without statuses
		getOptions = &pb.GetOptions{Start: 1, Stop: 2, ExcludeStatuses: true}
		instanceList, err = testCase.client.Get(context.Background(), getOptions)
		if err != nil {
			t.Fatal(err)
		}
		if len(instanceList.Instances) != 2 {
			t.Errorf("Expected a list of 2, got %d", len(instanceList.Instances))
		}
	})
}

func sentVSReturned(t *testing.T, returnedInstance, instance *pb.Instance) {
	if returnedInstance.Name != instance.Name {
		t.Error("Returned Instance Name differs from sent Instance Name")
	}
	if returnedInstance.KustomizationYaml != instance.KustomizationYaml {
		t.Error("Returned Instance YAML differs from sent Instance YAML")
	}
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

func generateRandomName(name string) string {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	randomName := []byte(fmt.Sprintf("%s-", name))
	for i := 0; i < 10; i++ {
		randomName = append(randomName, letters[rand.Intn(len(letters))])
	}
	return string(randomName)
}

var (
	deploymentYaml = []byte(`kind: Deployment
metadata:
  name: my-product
  namespace: default
  labels:
    app: my-product
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 2
  template:
    metadata:
      labels:
        app: my-product
    spec:
      restartPolicy: Always
      containers:
      - name: sleeper
        image: alpine:latest
        imagePullPolicy: IfNotPresent
        command:
          - sleep
        args:
          - '50000'
`)
)
