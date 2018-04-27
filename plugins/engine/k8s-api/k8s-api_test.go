package main

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/plugins/engine/k8s-api"
	"github.com/snarlysodboxer/hambone/plugins/render"
	"github.com/snarlysodboxer/hambone/plugins/state"
	"k8s.io/client-go/kubernetes/fake"
	"plugin"
	"testing"
)

const (
	renderPluginPath     = "../../render/default/default.so"
	stateStorePluginPath = "../../state/memory/memory.so"
	// deploymentTemplate = `apiVersion: apps/v1
	deploymentTemplate = `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: {{.Name}}
  namespace: default
  labels:
    app: {{.Name}}
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
        app: {{.Name}}
    spec:
      restartPolicy: Always
      containers:
      - name: sleeper
        image: my-account/my-repo:CHANGEME
        imagePullPolicy: IfNotPresent
        command:
        - sleep
        args:
        - '50000'
`
	serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.Name}}
  namespace: default
  labels:
    app: {{.Name}}
spec:
  selector:
    app: {{.Name}}
  type: ClusterIP
  ports:
  - name: server
    port: 80
    targetPort: server
    protocol: TCP
`
)

func newK8sEngine(rendererPath, storePath string) *main.K8sAPIEngine {
	stateStorePlugin, err := plugin.Open(storePath)
	if err != nil {
		panic(err)
	}
	stateStoreSymbol, err := stateStorePlugin.Lookup("StateStore")
	if err != nil {
		panic(err)
	}
	stateStore, ok := stateStoreSymbol.(state.Interface)
	if !ok {
		panic("Unexpected type from StateStore plugin")
	}

	renderPlugin, err := plugin.Open(rendererPath)
	if err != nil {
		panic(err)
	}
	rendererSymbol, err := renderPlugin.Lookup("Renderer")
	if err != nil {
		panic(err)
	}
	renderer, ok := rendererSymbol.(render.Interface)
	if !ok {
		panic("Unexpected type from Renderer plugin")
	}
	renderer.SetStateStore(stateStore)
	engine := main.NewK8sAPIEngine(renderer)
	engine.ClientSet = fake.NewSimpleClientset()

	return engine
}

func getStateStorePlugin(filePath string) state.Interface {
	stateStorePlugin, err := plugin.Open(filePath)
	if err != nil {
		panic(err)
	}
	stateStoreSymbol, err := stateStorePlugin.Lookup("StateStore")
	if err != nil {
		panic(err)
	}
	store, ok := stateStoreSymbol.(state.Interface)
	if !ok {
		panic("Unexpected type from StateStore plugin")
	}
	store.Init()
	return store
}

func getSpecGroup(name string) *pb.SpecGroup {
	spec := &pb.Spec{Name: "Deployment", Template: deploymentTemplate}
	specs := []*pb.Spec{}
	specs = append(specs, spec)
	spec = &pb.Spec{Name: "Service", Template: serviceTemplate}
	specs = append(specs, spec)
	specGroup := &pb.SpecGroup{}
	specGroup.Name = name
	specGroup.Specs = specs
	return specGroup
}

func getInstance(name string) *pb.Instance {
	instance := &pb.Instance{
		Name: name, SpecGroupName: "my-product", ValueSets: []*pb.ValueSet{
			&pb.ValueSet{"Deployment", `{"Name": "my-client"}`},
			&pb.ValueSet{"Service", `{"Name": "my-client"}`},
		},
	}
	return instance
}

func TestApplyInstance(t *testing.T) {
	store := getStateStorePlugin(stateStorePluginPath)
	specGroup := getSpecGroup("my-product")
	_, err := store.CreateSpecGroup(specGroup)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	instance := getInstance("my-client")
	_, err = store.CreateInstance(instance)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	engine := newK8sEngine(renderPluginPath, stateStorePluginPath)

	// Creates object in Kubernetes API
	err = engine.ApplyInstance(instance)
	if err != nil {
		t.Error(err)
	}

	// Updates instead of creates if the object already exists
	err = engine.ApplyInstance(instance)
	if err != nil {
		t.Error(err)
	}
}

// func TestDelete(t *testing.T) {
//     store := getStateStorePlugin(stateStorePluginPath)
//     specGroup := getSpecGroup("my-product")
//     _, err := store.CreateSpecGroup(specGroup)
//     if err != nil {
//         t.Error(err)
//         t.FailNow()
//     }
//     instance := getInstance("my-client")
//     _, err = store.CreateInstance(instance)
//     if err != nil {
//         t.Error(err)
//         t.FailNow()
//     }
//     engine := newK8sEngine(renderPluginPath, stateStorePluginPath)
//     err = engine.ApplyInstance(instance)
//     if err != nil {
//         t.Error(err)
//     }

//     // Deletes
//     // err = engine.DeleteInstance(instance)
//     // if err != nil {
//     //     t.Error(err)
//     // }
// }
