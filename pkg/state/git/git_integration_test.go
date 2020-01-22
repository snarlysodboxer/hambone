// +build integration

package git

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/helpers"
)

var (
	gitRepoAddress      = flag.String("git_repo_address", "http://localhost:5000/hambone/test-hambone.git", "The Git clone address for testing against")
	testEtcdLocksGitKey = flag.String("etcd_locks_with_git_key", "", "Key for etcd locks when using the Git state_store adapter. Leave unset to disable etcd locks with Git. Ignored with the etcd state_store adapter.")
	etcdEndpoints       = flag.String("etcd_endpoints", "http://127.0.0.1:2379", "Comma-separated list of etcd endpoints, only used for etcd adapter")
)

func TestGitUpdater(t *testing.T) {
	// setup
	tempDir, repoDir, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir) // clean up

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
	instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, instanceName)
	updater := &gitUpdater{instance, instanceDir, instanceFile, repoDir, nil}
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

	// check results
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
	err = ensureCleanFile(instanceDir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGitGetter(t *testing.T) {
	// setup
	tempDir, repoDir, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir) // clean up
	instancesDir := "overlays"
	subAppName := "my-app"
	dYamlName := filepath.Join(subAppName, "deployment.yaml")
	subSubAppName := filepath.Join(subAppName, "sub-dir")
	subYamlName := filepath.Join(subSubAppName, "statefulSet.yaml")
	file := &pb.File{Name: dYamlName, Directory: subAppName, Contents: string(deploymentYaml)}
	file2 := &pb.File{Name: subYamlName, Directory: subSubAppName, Contents: string(deploymentYaml)}
	sentList := &pb.InstanceList{}
	for i := 0; i < 10; i++ {
		instanceName := fmt.Sprintf("my-overlay-%d", i)
		instance := &pb.Instance{Name: instanceName, KustomizationYaml: string(kustomizationYaml), Files: []*pb.File{file, file2}}
		instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, instanceName)
		updater := &gitUpdater{instance, instanceDir, instanceFile, repoDir, nil}
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
		sentList.Instances = append(sentList.Instances, instance)
		err = updater.RunCleanupFuncs()
		if err != nil {
			t.Fatal(err)
		}
	}

	// get one
	getList := &pb.InstanceList{}
	getOptions := &pb.GetOptions{Name: "my-overlay-0"}
	getter := &gitGetter{getOptions, getList, instancesDir, repoDir, nil}
	defer func() {
		err := getter.RunCleanupFuncs()
		if err != nil {
			t.Fatal(err)
		}
	}()
	err = getter.Run()
	if err != nil {
		t.Fatal(err)
	}
	if len(getList.Instances) != 1 {
		t.Errorf("Expected list length of 1, got: %d", len(getList.Instances))
	}
	if getList.Instances[0].Name != "my-overlay-0" {
		t.Errorf("Expected Name my-overlay-0, got: %s", getList.Instances[0].Name)
	}
	if getList.Instances[0].KustomizationYaml != string(kustomizationYaml) {
		t.Errorf("Expected: %s\n, got: %s\n", string(kustomizationYaml), getList.Instances[0].KustomizationYaml)
	}
	if !reflect.DeepEqual(getList.Instances[0].Files[0], file) {
		t.Errorf("Expected: %s\n, got: %s\n", file, getList.Instances[0].Files[0])
	}
	err = getter.RunCleanupFuncs()
	if err != nil {
		t.Fatal(err)
	}

	// get all
	getList = &pb.InstanceList{}
	getOptions = &pb.GetOptions{}
	getter = &gitGetter{getOptions, getList, instancesDir, repoDir, nil}
	defer func() {
		err := getter.RunCleanupFuncs()
		if err != nil {
			t.Fatal(err)
		}
	}()
	err = getter.Run()
	if err != nil {
		t.Fatal(err)
	}
	for _, sentInst := range sentList.Instances {
		for _, getInst := range getList.Instances {
			if getInst.Name == sentInst.Name {
				if !reflect.DeepEqual(getInst, sentInst) {
					t.Errorf("Expected: %s\n, Got: %s\n", sentInst, getInst)
				}
			}
		}
	}
	err = getter.RunCleanupFuncs()
	if err != nil {
		t.Fatal(err)
	}

	// get paginated
	getList = &pb.InstanceList{}
	getOptions = &pb.GetOptions{Start: 2, Stop: 6, ExcludeStatuses: true}
	getter = &gitGetter{getOptions, getList, instancesDir, repoDir, nil}
	defer func() {
		err := getter.RunCleanupFuncs()
		if err != nil {
			t.Fatal(err)
		}
	}()
	err = getter.Run()
	if err != nil {
		t.Fatal(err)
	}
	if len(getList.Instances) != 5 {
		t.Errorf("Expected list length of 5, got: %d", len(getList.Instances))
	}
}

func TestGitTemplatesGetter(t *testing.T) {
	// setup
	tempDir, repoDir, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir) // clean up
	templatesDir := filepath.Join(repoDir, "templates")
	templateName := "my-template"
	// templates/my-template
	templateDir := filepath.Join(templatesDir, templateName)
	appName := "my-app"
	// templates/my-template/my-app
	myAppDir := filepath.Join(templateDir, appName)
	subAppName := "my-sub-app"
	// templates/my-template/my-app/my-sub-app
	myNestedDir := filepath.Join(myAppDir, subAppName)
	err = os.MkdirAll(myNestedDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	// templates/my-template/kustomization.yaml
	kYamlPath := filepath.Join(templateDir, helpers.KustomizationFileName)
	err = ioutil.WriteFile(kYamlPath, kustomizationYaml, 0666)
	if err != nil {
		t.Fatal(err)
	}
	dYamlName := "deployment.yaml"
	// templates/my-template/my-app/deployment.yaml
	dYamlPath := filepath.Join(myAppDir, dYamlName)
	err = ioutil.WriteFile(dYamlPath, deploymentYaml, 0666)
	if err != nil {
		t.Fatal(err)
	}
	// templates/my-template/my-app/my-sub-app/deployment.yaml
	dNestedYamlPath := filepath.Join(myNestedDir, dYamlName)
	err = ioutil.WriteFile(dNestedYamlPath, deploymentYaml, 0666)
	if err != nil {
		t.Fatal(err)
	}
	// templates/my-template/my-app/my-sub-app/kustomization.yaml
	kNestedYamlPath := filepath.Join(myNestedDir, helpers.KustomizationFileName)
	err = ioutil.WriteFile(kNestedYamlPath, kustomizationYaml, 0666)
	if err != nil {
		t.Fatal(err)
	}
	templatesGetter := &gitTemplatesGetter{&pb.InstanceList{}, templatesDir, repoDir, nil}
	defer func() {
		err := templatesGetter.RunCleanupFuncs()
		if err != nil {
			t.Fatal(err)
		}
	}()
	err = templatesGetter.Run()
	if err != nil {
		t.Fatal(err)
	}

	// check results
	template := templatesGetter.InstanceList.Instances[0]
	if len(template.Files) != 3 {
		t.Fatalf("Expected: %d, Got: %d", 3, len(template.Files))
	}
	if template.Name != templateName {
		t.Errorf("Expected: %s, Got: %s", templateName, template.Name)
	}
	if template.KustomizationYaml != string(kustomizationYaml) {
		t.Errorf("Expected: \n%s\n, Got: \n%s", string(kustomizationYaml), template.KustomizationYaml)
	}
	appFile := &pb.File{}
	// my-app/deployment.yaml
	dYamlFilePath := fmt.Sprintf("%s/%s", appName, dYamlName)
	subAppFile := &pb.File{}
	// my-app/my-sub-app
	subAppDir := fmt.Sprintf("%s/%s", appName, subAppName)
	// my-app/my-sub-app/deployment.yaml
	dNestedYamlFilePath := fmt.Sprintf("%s/%s", subAppDir, dYamlName)
	subAppKFile := &pb.File{}
	// my-app/my-sub-app/kustomization.yaml
	kNestedYamlFilePath := fmt.Sprintf("%s/%s", subAppDir, helpers.KustomizationFileName)
	for _, file := range template.Files {
		if file.Name == dYamlFilePath {
			appFile = file
		}
		if file.Name == dNestedYamlFilePath {
			subAppFile = file
		}
		if file.Name == kNestedYamlFilePath {
			subAppKFile = file
		}
	}
	if appFile.Name != dYamlFilePath {
		t.Errorf("Expected: %s, Got: %s", dYamlFilePath, appFile.Name)
	}
	if appFile.Directory != appName {
		t.Errorf("Expected: %s, Got: %s", appName, appFile.Directory)
	}
	if appFile.Contents != string(deploymentYaml) {
		t.Errorf("Expected: \n%s\n, Got: \n%s", string(deploymentYaml), appFile.Contents)
	}
	if subAppFile.Name != dNestedYamlFilePath {
		t.Errorf("Expected: %s, Got: %s", dNestedYamlFilePath, subAppFile.Name)
	}
	if subAppFile.Directory != subAppDir {
		t.Errorf("Expected: %s, Got: %s", subAppDir, subAppFile.Directory)
	}
	if subAppFile.Contents != string(deploymentYaml) {
		t.Errorf("Expected: \n%s\n, Got: \n%s", string(deploymentYaml), subAppFile.Contents)
	}
	if subAppKFile.Name != kNestedYamlFilePath {
		t.Errorf("Expected: %s, Got: %s", kNestedYamlFilePath, subAppKFile.Name)
	}
	if subAppKFile.Directory != subAppDir {
		t.Errorf("Expected: %s, Got: %s", subAppDir, subAppKFile.Directory)
	}
	if subAppKFile.Contents != string(kustomizationYaml) {
		t.Errorf("Expected: \n%s\n, Got: \n%s", string(kustomizationYaml), subAppKFile.Contents)
	}
}

func TestGitDeleter(t *testing.T) {
	// setup
	tempDir, repoDir, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir) // clean up

	instanceName := "my-overlay-delete"
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
	instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, instanceName)
	updater := &gitUpdater{instance, instanceDir, instanceFile, repoDir, nil}
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
	deleter := &gitDeleter{instance, instanceDir, instanceFile, repoDir, nil}
	err = deleter.Init()
	defer func() {
		err := deleter.RunCleanupFuncs()
		if err != nil {
			t.Fatal(err)
		}
	}()
	if err != nil {
		t.Fatal(err)
	}
	err = deleter.Commit()
	if err != nil {
		t.Fatal(err)
	}

	// check results
	_, err = ioutil.ReadFile(instanceFile)
	if err == nil {
		t.Errorf("Expected error reading %s file, got none", instanceFile)
	}
	_, err = ioutil.ReadFile(dYamlPath)
	if err == nil {
		t.Errorf("Expected error reading %s file, got none", dYamlPath)
	}
	_, err = ioutil.ReadFile(subYamlPath)
	if err == nil {
		t.Errorf("Expected error reading %s file, got none", subYamlPath)
	}
	if _, err := os.Stat(instanceDir); !os.IsNotExist(err) {
		t.Errorf("Expected %s to not exist, found it", instanceDir)
	}
	err = ensureCleanFile(instanceDir)
	if err != nil {
		t.Error(err)
	}
}

func setup() (tempDir, repoDir string, err error) {
	etcdLocksGitKey = *testEtcdLocksGitKey
	tempDir, err = ioutil.TempDir("", "hambone-git")
	if err != nil {
		return
	}
	err = os.Chdir(tempDir)
	if err != nil {
		return
	}
	args := []string{`clone`, *gitRepoAddress}
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%s: %s", err.Error(), string(output))
		return
	}
	repoDir = filepath.Join(tempDir, "test-hambone")
	stateStore := &Engine{WorkingDir: repoDir, EndpointsString: *etcdEndpoints, EtcdLocksGitKey: etcdLocksGitKey}
	err = stateStore.Init()
	if err != nil {
		return
	}

	return
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
