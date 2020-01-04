// +build integration

package etcd

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/helpers"
	"go.etcd.io/etcd/clientv3"

	"context"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	etcdEndpoints = flag.String("etcd_endpoints", "http://127.0.0.1:2379", "Comma-separated list of etcd endpoints, only used for etcd adapter")
)

func TestEtcdUpdater(t *testing.T) {
	engine, clientV3, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = clientV3.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	instanceName := "my-overlay"
	instancesDir := "overlays"
	subAppName := "my-app"
	dYamlName := filepath.Join(subAppName, "deployment.yaml")
	subSubAppName := filepath.Join(subAppName, "sub-dir")
	subYamlName := filepath.Join(subSubAppName, "statefulSet.yaml")
	dYamlPath := filepath.Join(instancesDir, instanceName, dYamlName)
	subYamlPath := filepath.Join(instancesDir, instanceName, subYamlName)
	file := &pb.File{Name: dYamlName, Directory: subAppName, Contents: string(deploymentYaml)}
	file2 := &pb.File{Name: subYamlName, Directory: subSubAppName, Contents: string(deploymentYaml)}
	instance := &pb.Instance{Name: instanceName, KustomizationYaml: string(kustomizationYaml), Files: []*pb.File{file, file2}}
	updater := engine.NewUpdater(instance, instancesDir)
	err = updater.Init()
	defer func() {
		err := updater.RunCleanupFuncs()
		if err != nil {
			t.Fatal(err)
		}
	}()
	if err != nil {
		t.Fatal(err)
	}
	err = updater.Commit()
	if err != nil {
		t.Fatal(err)
	}
	err = updater.RunCleanupFuncs()
	if err != nil {
		t.Fatal(err)
	}

	// check results
	kvClient := clientv3.NewKV(clientV3)
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	response := &clientv3.GetResponse{}
	instanceKey := getInstanceKey(instanceName)
	response, err = kvClient.Get(ctx, instanceKey)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	returnedName := stripInstanceKeyPrefix(string(response.Kvs[0].Key))
	if returnedName != instanceName {
		t.Errorf("Expected %s, got %s", instanceName, returnedName)
	}
	returnedValue := string(response.Kvs[0].Value)
	if returnedValue != string(kustomizationYaml) {
		t.Errorf("Expected %s\n, got %s\n", string(kustomizationYaml), returnedValue)
	}
	ctx, cancel = context.WithTimeout(context.Background(), requestTimeout)
	response = &clientv3.GetResponse{}
	response, err = kvClient.Get(ctx, getFileKey(instanceKey, dYamlName))
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if len(response.Kvs) != 1 {
		t.Fatalf("Expected 1, got %d", len(response.Kvs))
	}
	returnedName = stripInstanceKeyPrefix(string(response.Kvs[0].Key))
	if returnedName != filepath.Join(instanceName, dYamlName) {
		t.Errorf("Expected %s, got %s", filepath.Join(instanceName, dYamlName), returnedName)
	}
	returnedValue = string(response.Kvs[0].Value)
	if returnedValue != string(deploymentYaml) {
		t.Errorf("Expected %s\n, got %s\n", string(deploymentYaml), returnedValue)
	}
	_, instanceFile := helpers.GetInstanceDirFile(instancesDir, instanceName)
	contents, err := ioutil.ReadFile(instanceFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(kustomizationYaml) {
		t.Fatal(err)
	}
	contents, err = ioutil.ReadFile(dYamlPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(deploymentYaml) {
		t.Fatal(err)
	}
	contents, err = ioutil.ReadFile(subYamlPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(deploymentYaml) {
		t.Fatal(err)
	}
}

func setup() (*Engine, *clientv3.Client, error) {
	stateStore := &Engine{}
	endpoints := strings.Split(*etcdEndpoints, ",")
	clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
	if err != nil {
		return stateStore, clientV3, err
	}
	tempDir, err := ioutil.TempDir("", "hambone-etcd")
	if err != nil {
		return stateStore, clientV3, err
	}
	err = os.Chdir(tempDir)
	if err != nil {
		return stateStore, clientV3, err
	}
	repoDir := filepath.Join(tempDir, "test-hambone")
	err = os.MkdirAll(repoDir, 0755)
	if err != nil {
		return stateStore, clientV3, err
	}
	stateStore = &Engine{WorkingDir: repoDir, EndpointsString: *etcdEndpoints}
	err = stateStore.Init()
	if err != nil {
		return stateStore, clientV3, err
	}

	return stateStore, clientV3, nil
}

var (
	kustomizationYaml = []byte(`---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../../base

configMapGenerator:
- name: my-configmap
  namespace: default
  behavior: create
  literals:
  - APP_URL=https://asdf.example.com
  - SOME=thing
`)
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
