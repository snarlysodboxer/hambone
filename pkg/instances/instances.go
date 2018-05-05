package instances

import (
	"bytes"
	"errors"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/helpers"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"os/exec"
	"strings"
)

// TODO log errors

type GetRequest struct {
	*pb.GetOptions
	*pb.InstanceList
	server *InstancesServer
}

func NewGetRequest(getOptions *pb.GetOptions, server *InstancesServer) *GetRequest {
	return &GetRequest{getOptions, &pb.InstanceList{}, server}
}

func (request *GetRequest) Run() error {
	list := &pb.InstanceList{}

	getter := request.server.StateStore.NewGetter(request.GetOptions, list, request.server.InstancesDir)
	err := getter.Run()
	if err != nil {
		return err
	}

	// load statuses if desired
	if !request.GetOptions.GetExcludeStatuses() {
		for _, pbInstance := range list.Instances {
			instance := NewInstance(pbInstance, request.server)
			_ = instance.loadStatuses()
		}
	}
	request.InstanceList = list

	return nil
}

type Instance struct {
	*pb.Instance
	InstanceDir  string
	InstanceFile string
	server       *InstancesServer
}

func NewInstance(pbInstance *pb.Instance, server *InstancesServer) *Instance {
	instanceDir := fmt.Sprintf(`%s/%s`, server.InstancesDir, pbInstance.Name)
	instanceFile := fmt.Sprintf(`%s/%s`, instanceDir, helpers.KustomizationFileName)
	return &Instance{pbInstance, instanceDir, instanceFile, server}
}

func (instance *Instance) apply() error {
	// ensure namePrefix in yaml matches pb.Instance.Name
	err := namePrefixMatches(instance.Instance)
	if err != nil {
		return err
	}

	updater := instance.server.StateStore.NewUpdater(instance.Instance, instance.server.InstancesDir)
	err = updater.Init()
	if err != nil {
		return err
	}

	// kustomize build <instance path> | kubctl apply -f -
	_, err = instance.pipeKustomizeToKubectl(false, `apply`, `-f`, `-`)
	if err != nil {
		return updater.Cancel(err)
	}

	err = updater.Commit()
	if err != nil {
		return err
	}

	// fill in statuses
	_ = instance.loadStatuses()

	return nil
}

func (instance *Instance) delete() error {
	deleter := instance.server.StateStore.NewDeleter(instance.Instance, instance.server.InstancesDir)
	err := deleter.Init()
	if err != nil {
		return err
	}

	// kustomize build <instance path> | kubctl delete -f -
	_, err = instance.pipeKustomizeToKubectl(false, `delete`, `-f`, `-`)
	if err != nil {
		return deleter.Cancel(err)
	}

	err = deleter.Commit()
	if err != nil {
		return err
	}

	return nil
}

type ItemStatuses struct {
	Items []Item `yaml:"items"`
}

type Item struct {
	Kind     string            `yaml:"kind"`
	Metadata Metadata          `yaml:"metadata"`
	Labels   map[string]string `yaml:"labels"`
	Status   ItemStatus        `yaml:"status"`
}

type Metadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type ItemStatus struct {
	AvailableReplicas int32 `yaml:"availableReplicas"`
	ReadyReplicas     int32 `yaml:"readyReplicas"`
	DesiredReplicas   int32 `yaml:"replicas"`
	UpdatedReplicas   int32 `yaml:"updatedReplicas"`
}

func (instance *Instance) loadStatuses() error {
	output, err := instance.pipeKustomizeToKubectl(true, `get`, `-o`, `yaml`, `-f`, `-`)
	if err != nil {
		instance.Instance.StatusesErrorMessage = string(output)
		return err
	}
	items := ItemStatuses{}
	if err = yaml.Unmarshal([]byte(output), &items); err != nil {
		instance.Instance.StatusesErrorMessage = string(output)
		return err
	}
	for _, item := range items.Items {
		switch item.Kind {
		case "Deployment":
			deploymentStatus := &pb.DeploymentStatus{}
			deploymentStatus.Name = item.Metadata.Name
			deploymentStatus.Desired = item.Status.DesiredReplicas
			deploymentStatus.Current = item.Status.ReadyReplicas
			deploymentStatus.Available = item.Status.AvailableReplicas
			deploymentStatus.UpToDate = item.Status.UpdatedReplicas
			statusDeployment := &pb.Status_Deployment{deploymentStatus}
			status := &pb.Status{Item: statusDeployment}
			instance.Instance.Statuses = append(instance.Instance.Statuses, status)
		}
		// TODO more cases here
	}
	return nil
}

func (instance *Instance) pipeKustomizeToKubectl(suppressOutput bool, args ...string) ([]byte, error) {
	instanceDir := instance.InstanceDir
	emptybytes := []byte{}

	kustomizeCmd := exec.Command("kustomize", "build", instanceDir)
	stdout, err := kustomizeCmd.StdoutPipe()
	if err != nil {
		return emptybytes, err
	}
	stderr, err := kustomizeCmd.StderrPipe()
	if err != nil {
		return emptybytes, err
	}
	if err := kustomizeCmd.Start(); err != nil {
		return emptybytes, err
	}
	kubectlCmd := exec.Command(`kubectl`, args...)
	stdin, err := kubectlCmd.StdinPipe()
	if err != nil {
		return emptybytes, err
	}
	go func() {
		defer stdin.Close()
		_, err = io.Copy(stdin, stdout)
		if err != nil {
			return // TODO think about this
		}
	}()
	buf := new(bytes.Buffer)
	buf.ReadFrom(stderr)
	if err := kustomizeCmd.Wait(); err != nil {
		helpers.PrintExecOutput(buf.Bytes(), "kustomize", `build`, instanceDir)
		return buf.Bytes(), errors.New(fmt.Sprintf("ERROR running `kustomize build %s`:\n%s%s", instanceDir, strings.TrimSuffix(buf.String(), "\n"), err.Error()))
	}
	output, err := kubectlCmd.CombinedOutput()
	if suppressOutput {
		helpers.Printf("Ran `kustomize build %s | kubectl %s` and got success\n\n", instanceDir, strings.Join(args, " "))
	} else {
		helpers.PrintExecOutput(output, fmt.Sprintf("kustomize build %s | kubectl", instanceDir), args...)
	}
	if err != nil {
		return output, err
	}

	return output, nil
}

type kustomizationYaml struct {
	NamePrefix string `yaml:"namePrefix"`
}

func namePrefixMatches(instance *pb.Instance) error {
	kYaml := kustomizationYaml{}
	err := yaml.Unmarshal([]byte(instance.KustomizationYaml), &kYaml)
	if err != nil {
		return err
	}
	hyphened := fmt.Sprintf("%s-", instance.Name)
	if kYaml.NamePrefix != hyphened {
		return errors.New(fmt.Sprintf("Instance Name does not match `namePrefix`, got: %s, want: %s", kYaml.NamePrefix, hyphened))
	}
	return nil
}

func isEmpty(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	_, err = file.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
