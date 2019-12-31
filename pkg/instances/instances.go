package instances

import (
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/helpers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"

	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
)

// TODO figure out why helpful error is not returned when the k8s API server is unavailable

var (
	// InstanceNameMismatchError indicates a request's Instance.Name and OldInstance.Name do not match
	InstanceNameMismatchError = status.Error(codes.FailedPrecondition, "Instance.Name and OldInstance.Name do not match")
)

// GetTemplatesRequest represents a get request
type GetTemplatesRequest struct {
	*pb.InstanceList
	server *Server
}

// NewGetTemplatesRequest returns an instantiated GetTemplatesRequest
func NewGetTemplatesRequest(server *Server) *GetTemplatesRequest {
	return &GetTemplatesRequest{&pb.InstanceList{}, server}
}

// Run runs the GetTemplatesRequest
func (request *GetTemplatesRequest) Run() error {
	list := &pb.InstanceList{}

	getter := request.server.StateStore.NewTemplatesGetter(list, request.server.TemplatesDir)
	err := getter.Run()
	if err != nil {
		return err
	}

	request.InstanceList = list

	return nil
}

// GetRequest represents a get request
type GetRequest struct {
	*pb.GetOptions
	*pb.InstanceList
	server *Server
}

// NewGetRequest returns an instantiated GetRequest
func NewGetRequest(getOptions *pb.GetOptions, server *Server) *GetRequest {
	return &GetRequest{getOptions, &pb.InstanceList{}, server}
}

// Run runs the GetRequest
func (request *GetRequest) Run() error {
	list := &pb.InstanceList{}

	getter := request.server.StateStore.NewGetter(request.GetOptions, list, request.server.InstancesDir)
	err := getter.Run()
	if err != nil {
		return err
	}

	// load statuses if desired
	if request.server.EnableKubectl {
		if !request.GetOptions.GetExcludeStatuses() {
			for _, pbInstance := range list.Instances {
				instance := NewInstance(pbInstance, request.server)
				_ = instance.loadStatuses()
			}
		}
	}
	request.InstanceList = list

	return nil
}

// Instance represents an overlay Instance
type Instance struct {
	*pb.Instance
	InstanceDir  string
	InstanceFile string
	server       *Server
}

// NewInstance returns an instantiated Instance
func NewInstance(pbInstance *pb.Instance, server *Server) *Instance {
	instanceDir := fmt.Sprintf(`%s/%s`, server.InstancesDir, pbInstance.Name)
	instanceFile := fmt.Sprintf(`%s/%s`, instanceDir, helpers.KustomizationFileName)
	return &Instance{pbInstance, instanceDir, instanceFile, server}
}

func (instance *Instance) apply() error {
	err := NamesEquate(instance)
	if err != nil {
		return err
	}

	updater := instance.server.StateStore.NewUpdater(instance.Instance, instance.server.InstancesDir)
	err = updater.Init()
	if err != nil {
		return err
	}

	switch {
	case instance.server.EnableKubectl:
		// kustomize build <instance path> | kubctl apply -f -
		_, err = instance.pipeKustomizeToKubectl(false, `apply`, `-f`, `-`)
		if err != nil {
			return updater.Cancel(err)
		}
	case instance.server.EnableKustomizeBuild:
		// kustomize build <instance path>
		err = instance.kustomizeBuild()
		if err != nil {
			return updater.Cancel(err)
		}
	}

	err = updater.Commit()
	if err != nil {
		return err
	}

	if instance.server.EnableKubectl {
		// fill in statuses
		_ = instance.loadStatuses()
	}

	// don't return OldInstance in response
	instance.Instance.OldInstance = nil

	return nil
}

func (instance *Instance) delete() error {
	err := NamesEquate(instance)
	if err != nil {
		return err
	}

	deleter := instance.server.StateStore.NewDeleter(instance.Instance, instance.server.InstancesDir)
	err = deleter.Init()
	if err != nil {
		return err
	}

	if instance.server.EnableKubectl {
		// kustomize build <instance path> | kubctl delete -f -
		_, err = instance.pipeKustomizeToKubectl(false, `delete`, `-f`, `-`)
		if err != nil {
			return deleter.Cancel(err)
		}
	}

	err = deleter.Commit()
	if err != nil {
		return err
	}

	// don't return OldInstance in response
	instance.Instance.OldInstance = nil

	return nil
}

// ItemStatuses represents a list of ItemStatuses
type ItemStatuses struct {
	Items []Item `yaml:"items"`
}

// Item represents a Kubernetes Object
type Item struct {
	Kind     string            `yaml:"kind"`
	Metadata Metadata          `yaml:"metadata"`
	Labels   map[string]string `yaml:"labels"`
	Status   ItemStatus        `yaml:"status"`
}

// Metadata represents a Kubernetes Object's Metadata
type Metadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// ItemStatus respresents the status of a Kubernetes Object
type ItemStatus struct {
	AvailableReplicas int32 `yaml:"availableReplicas"`
	ReadyReplicas     int32 `yaml:"readyReplicas"`
	DesiredReplicas   int32 `yaml:"replicas"`
	UpdatedReplicas   int32 `yaml:"updatedReplicas"`
}

func (instance *Instance) loadStatuses() error {
	output, err := instance.pipeKustomizeToKubectl(true, `get`, `-o`, `yaml`, `-f`, `-`)
	if err != nil {
		log.Println(err)
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
			statusDeployment := &pb.Status_Deployment{Deployment: deploymentStatus}
			status := &pb.Status{Item: statusDeployment}
			instance.Instance.Statuses = append(instance.Instance.Statuses, status)
		}
		// TODO more cases here
	}
	return nil
}

func (instance *Instance) pipeKustomizeToKubectl(suppressOutput bool, args ...string) ([]byte, error) {
	instanceDir := instance.InstanceDir
	emptyBytes := []byte{}

	// run kustomize build
	kustomizeCmd := exec.Command("kustomize", "build", instanceDir)
	stdout, err := kustomizeCmd.StdoutPipe()
	if err != nil {
		log.Println(err)
		return emptyBytes, err
	}
	stderr, err := kustomizeCmd.StderrPipe()
	if err != nil {
		log.Println(err)
		return emptyBytes, err
	}
	if err := kustomizeCmd.Start(); err != nil {
		log.Println(err)
		return emptyBytes, err
	}

	// check for empty stdout from kustomizeCmd
	buffer := new(bytes.Buffer)
	buffer.ReadFrom(stdout)
	stdout = ioutil.NopCloser(bytes.NewBuffer(buffer.Bytes()))
	if buffer.String() == "" {
		msg := fmt.Sprintf("No output from `kustomize build %s`", instanceDir)
		err = errors.New(msg)
		log.Println(err)
		return []byte(msg), err
	}

	// prepare kubectlCmd
	kubectlCmd := exec.Command(`kubectl`, args...)
	stdin, err := kubectlCmd.StdinPipe()
	if err != nil {
		log.Println(err)
		return emptyBytes, err
	}

	// in a separate thread, prepare to copy kustomizeCmd stdout to kubectlCmd stdin
	go func() {
		defer stdin.Close()
		_, err = io.Copy(stdin, stdout)
		if err != nil {
			log.Println(err)
			return // TODO think about this
		}
	}()

	// read kustomizeCmd stderr in case there was an error
	buffer = new(bytes.Buffer)
	buffer.ReadFrom(stderr)

	// ensure kustomizeCmd has completed
	if err := kustomizeCmd.Wait(); err != nil {
		helpers.DebugExecOutput(buffer.Bytes(), "kustomize", `build`, instanceDir)
		return buffer.Bytes(), fmt.Errorf("ERROR running `kustomize build %s`:\n%s%s", instanceDir, strings.TrimSuffix(buffer.String(), "\n"), err.Error())
	}

	// pipe kustomizeCmd into kubectlCmd
	output, err := kubectlCmd.CombinedOutput()
	if suppressOutput {
		helpers.Debugf("Ran `kustomize build %s | kubectl %s` and got success\n\n", instanceDir, strings.Join(args, " "))
	} else {
		helpers.DebugExecOutput(output, fmt.Sprintf("kustomize build %s | kubectl", instanceDir), args...)
	}
	if err != nil {
		return output, err
	}

	return output, nil
}

func (instance *Instance) kustomizeBuild() error {
	instanceDir := instance.InstanceDir

	// run kustomize build
	kustomizeCmd := exec.Command("kustomize", "build", instanceDir)
	stdout, err := kustomizeCmd.StdoutPipe()
	if err != nil {
		log.Println(err)
		return err
	}
	stderr, err := kustomizeCmd.StderrPipe()
	if err != nil {
		log.Println(err)
		return err
	}
	if err := kustomizeCmd.Start(); err != nil {
		log.Println(err)
		return err
	}

	// check for empty stdout from kustomizeCmd
	buffer := new(bytes.Buffer)
	buffer.ReadFrom(stdout)
	stdout = ioutil.NopCloser(bytes.NewBuffer(buffer.Bytes()))
	if buffer.String() == "" {
		msg := fmt.Sprintf("No output from `kustomize build %s`", instanceDir)
		err = errors.New(msg)
		log.Println(err)
		return err
	}

	// read kustomizeCmd stderr in case there was an error
	buffer = new(bytes.Buffer)
	buffer.ReadFrom(stderr)

	// ensure kustomizeCmd has completed
	if err := kustomizeCmd.Wait(); err != nil {
		helpers.DebugExecOutput(buffer.Bytes(), "kustomize", `build`, instanceDir)
		return fmt.Errorf("ERROR running `kustomize build %s`:\n%s%s", instanceDir, strings.TrimSuffix(buffer.String(), "\n"), err.Error())
	}

	return nil
}

// NamesEquate ensures Instance Name and OldInstance Name are the same
func NamesEquate(instance *Instance) error {
	if instance.OldInstance != nil {
		if instance.Name != instance.OldInstance.Name {
			return InstanceNameMismatchError
		}
	}

	return nil
}
