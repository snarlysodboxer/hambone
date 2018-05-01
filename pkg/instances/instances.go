package instances

import (
	"bytes"
	"errors"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// NOTES for Git plugin:
// * Setup lock plugin, with an in-memory option, (etcd later)
//     * With in-memory option, you can only run one replica of the app at a time
//     * With etcd option, you can run muliple replicas of the app

// ensure repo is clean and pulled at init time (push if needed)
// cluster-wide lock on any repo changes
// if update fails, reset repo, release lock, return error to caller
// if a committed push fails, retry and eventually, release lock, return error to caller, (shutdown, delete self-pod)?

// TODO periodically check that working tree is clean?

// TODO log errors

const (
	kustomizationFileName = "kustomization.yaml"
)

type getRequest struct {
	*pb.GetOptions
	*pb.InstanceList
	instancesDir string
}

func NewGetRequest(getOptions *pb.GetOptions, instancesDir string) *getRequest {
	return &getRequest{getOptions, &pb.InstanceList{}, instancesDir}
}

func (request *getRequest) Run() error {
	list := &pb.InstanceList{}

	// git pull
	output, err := exec.Command("git", "pull").CombinedOutput()
	if err != nil {
		return newExecError(err, output, "git", "pull")
	}

	// list instances directory, sort
	files, err := ioutil.ReadDir(request.instancesDir) // ReadDir sorts
	if err != nil {
		return err
	}
	// TODO DRY this out
	if request.GetOptions.GetName() != "" {
		for _, file := range files {
			if file.IsDir() {
				if file.Name() == request.GetOptions.GetName() {
					kFile := fmt.Sprintf("%s/%s/%s", request.instancesDir, file.Name(), kustomizationFileName)
					if _, err := os.Stat(kFile); os.IsNotExist(err) {
						return errors.New(fmt.Sprintf("Found directory `%s/%s` but it does not contain a `%s` file", request.instancesDir, file.Name(), kustomizationFileName))
					}

					contents, err := ioutil.ReadFile(kFile)
					if err != nil {
						return err
					}
					instance := &pb.Instance{Name: file.Name(), KustomizationYaml: string(contents)}
					list.Instances = append(list.Instances, instance)
				}
			}
		}
	} else {
		for _, file := range files {
			if file.IsDir() {
				kFile := fmt.Sprintf("%s/%s/%s", request.instancesDir, file.Name(), kustomizationFileName)
				if _, err := os.Stat(kFile); os.IsNotExist(err) {
					debug("WARNING found directory `%s/%s` that does not contain a `%s` file, skipping", request.instancesDir, file.Name(), kustomizationFileName)
					continue
				}
				contents, err := ioutil.ReadFile(kFile)
				if err != nil {
					return err
				}
				instance := &pb.Instance{Name: file.Name(), KustomizationYaml: string(contents)}
				list.Instances = append(list.Instances, instance)
			}
		}

		// filter list to start and stop points in getOptions
		indexStart, indexStop := ConvertStartStopToSliceIndexes(request.GetOptions.GetStart(), request.GetOptions.GetStop(), int32(len(list.Instances)))
		if indexStop == 0 {
			list.Instances = list.Instances[indexStart:]
		} else {
			list.Instances = list.Instances[indexStart:indexStop]
		}
	}

	// load statuses if desired
	if !request.GetOptions.GetExcludeStatuses() {
		for _, pbInstance := range list.Instances {
			instance := NewInstance(pbInstance, request.instancesDir)
			_ = instance.loadStatuses()
		}
	}
	request.InstanceList = list

	return nil
}

type instance struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
}

func NewInstance(pbInstance *pb.Instance, instancesDir string) *instance {
	instanceDir := fmt.Sprintf(`%s/%s`, instancesDir, pbInstance.Name)
	instanceFile := fmt.Sprintf(`%s/%s`, instanceDir, kustomizationFileName)
	return &instance{pbInstance, instanceDir, instanceFile}
}

func (instance *instance) apply() error {
	instanceDir := instance.instanceDir
	instanceFile := instance.instanceFile

	// ensure namePrefix in yaml matches pb.Instance.Name
	err := namePrefixMatches(instance.Instance)
	if err != nil {
		return err
	}

	// ensure instance file is clean in git
	// check for tracked changes
	args := []string{`diff`, `--exit-code`, `--`, instanceFile}
	output, err := exec.Command("git", args...).CombinedOutput()
	debugExecOutput(output, "git", args...)
	if err != nil {
		return errors.New(fmt.Sprintf("There are tracked uncommitted changes for this Instance! This should not happen and could indicate a bug. Fix this manually:\n%s\n", indent(output)))
	}
	// check for untracked changes
	test := fmt.Sprintf("git ls-files --exclude-standard --others %s", instanceFile)
	args = []string{`-c`, fmt.Sprintf("test -z $(%s)", test)}
	output, err = exec.Command("sh", args...).CombinedOutput()
	debugExecOutput(output, "sh", args...)
	if err != nil {
		return errors.New("There are untracked uncommitted changes for this Instance! This should not happen and could indicate a bug. Fix this manually.")
	}

	// git pull
	output, err = exec.Command("git", "pull").CombinedOutput()
	if err != nil {
		return newExecError(err, output, "git", "pull")
	}

	// mkdir
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return err
	}

	// write <instancesDir>/<name>/kustomization.yml
	if err := ioutil.WriteFile(instanceFile, []byte(instance.KustomizationYaml), 0644); err != nil {
		return err
	}
	debugf("Wrote `%s` with contents:\n\t%s\n", instanceFile, indent([]byte(instance.KustomizationYaml)))

	// kustomize build <instance path> | kubctl apply -f -
	_, err = instance.pipeKustomizeToKubectl(`apply`, `-f`, `-`)
	if err != nil {
		return err
	}

	// check if there's anything to commit
	args = []string{`diff`, `--exit-code`, instanceFile}
	output, err = exec.Command("git", args...).CombinedOutput()
	debugExecOutput(output, "git", args...)
	test = fmt.Sprintf("git ls-files --exclude-standard --others %s", instanceFile)
	args = []string{`-c`, fmt.Sprintf("test -z $(%s)", test)}
	output, untrackedErr := exec.Command("sh", args...).CombinedOutput()
	debugExecOutput(output, "sh", args...)
	if err != nil || untrackedErr != nil {
		// Changes to commit

		// git add
		args = []string{"add", instanceFile}
		if _, err := rollbackCommand(instanceDir, instanceFile, "git", args...); err != nil {
			return err
		}

		// git commit
		args = []string{"commit", "-m", fmt.Sprintf(`Automate hambone apply for %s`, instance.Name)}
		if _, err := rollbackCommand(instanceDir, instanceFile, "git", args...); err != nil {
			return err
		}

		// git push
		if _, err = rollbackCommand(instanceDir, instanceFile, "git", "push"); err != nil {
			return err
		}
	}

	// fill in statuses
	_ = instance.loadStatuses()

	return nil
}

func (instance *instance) delete() error {
	instanceFile := instance.instanceFile

	// TODO DRY this up

	// git pull
	output, err := exec.Command("git", "pull").CombinedOutput()
	if err != nil {
		return newExecError(err, output, "git", "pull")
	}

	// ensure Instance exists
	if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("ERROR Instance not found at `%s`", instanceFile))
	}

	// ensure Instance file is clean in git
	// check for tracked changes
	args := []string{`diff`, `--exit-code`, `--`, instanceFile}
	output, err = exec.Command("git", args...).CombinedOutput()
	debugExecOutput(output, "git", args...)
	if err != nil {
		return errors.New(fmt.Sprintf("There are tracked uncommitted changes for this Instance! This should not happen and could indicate a bug. Fix this manually:\n%s\n", indent(output)))
	}
	// check for untracked changes
	test := fmt.Sprintf("git ls-files --exclude-standard --others %s", instanceFile)
	args = []string{`-c`, fmt.Sprintf("test -z $(%s)", test)}
	output, err = exec.Command("sh", args...).CombinedOutput()
	debugExecOutput(output, "sh", args...)
	if err != nil {
		return errors.New("There are untracked uncommitted changes for this Instance! This should not happen and could indicate a bug. Fix this manually.")
	}

	// kustomize build <instance path> | kubctl delete -f -
	_, err = instance.pipeKustomizeToKubectl(`delete`, `-f`, `-`)
	if err != nil {
		return err
	}

	// TODO consider the case where any of the following fail, but the objects have been deleted from k8s

	// git rm <instancesDir>/<name>/kustomization.yaml
	output, err = exec.Command("git", "rm", instanceFile).CombinedOutput()
	if err != nil {
		return newExecError(err, output, "git", "rm", instanceFile)
	}
	debugExecOutput(output, "git", args...)

	// git commit
	args = []string{"commit", "-m", fmt.Sprintf(`Automate hambone delete for %s`, instance.Name)}
	output, err = exec.Command("git", args...).CombinedOutput()
	debugExecOutput(output, "git", args...)
	if err != nil {
		return errors.New(fmt.Sprintf("ERROR with `git %s`, retry manually!\n%s%s", strings.Join(args, " "), err.Error(), indent(output)))
	}

	// TODO think about retries here and other places
	// git push
	output, err = exec.Command("git", "push").CombinedOutput()
	debugExecOutput(output, "git", "push")
	if err != nil {
		return errors.New(fmt.Sprintf("ERROR with `git push`, retry manually!\n%s%s", err.Error(), indent(output)))
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

func (instance *instance) loadStatuses() error {
	output, err := instance.pipeKustomizeToKubectl(`get`, `-o`, `yaml`, `-f`, `-`)
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

func (instance *instance) pipeKustomizeToKubectl(args ...string) ([]byte, error) {
	instanceDir := instance.instanceDir
	instanceFile := instance.instanceFile
	emptybytes := []byte{}

	kustomizeCmd := exec.Command("kustomize", "build", instanceDir)
	stdout, err := kustomizeCmd.StdoutPipe()
	if err != nil {
		return emptybytes, rollbackAndError(instanceDir, instanceFile, err)
	}
	stderr, err := kustomizeCmd.StderrPipe()
	if err != nil {
		return emptybytes, rollbackAndError(instanceDir, instanceFile, err)
	}
	if err := kustomizeCmd.Start(); err != nil {
		return emptybytes, rollbackAndError(instanceDir, instanceFile, err)
	}
	kubectlCmd := exec.Command(`kubectl`, args...)
	stdin, err := kubectlCmd.StdinPipe()
	if err != nil {
		return emptybytes, rollbackAndError(instanceDir, instanceFile, err)
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
		debugExecOutput(buf.Bytes(), "kustomize", `build`, instanceDir)
		return emptybytes, rollbackAndError(instanceDir, instanceFile, errors.New(fmt.Sprintf("ERROR running `kustomize build %s`:\n%s%s", instanceDir, strings.TrimSuffix(buf.String(), "\n"), err.Error())))
	}
	output, err := kubectlCmd.CombinedOutput()
	debugExecOutput(output, "kubectl", args...)
	if err != nil {
		return output, rollbackAndError(instanceDir, instanceFile, err)
	}
	return output, nil
}

func rollbackCommand(instanceDir, instanceFile, cmd string, args ...string) ([]byte, error) {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	debugExecOutput(output, cmd, args...)
	if err != nil {
		return output, rollbackAndError(instanceDir, instanceFile, err)
	}
	return output, nil
}

// TODO untangle rollback functions
func rollbackAndError(instanceDir, instanceFile string, err error) error {
	rollbackErr := rollback(instanceDir, instanceFile)
	if rollbackErr != nil {
		return errors.New(fmt.Sprintf("ERROR rolling back!\n%s\n%s\n", indent([]byte(rollbackErr.Error())), err.Error()))
	}
	return err
}

func rollback(instanceDir, instanceFile string) error {
	if _, err := exec.Command("git", "ls-files", "--error-unmatch", instanceFile).CombinedOutput(); err != nil {
		// File is not tracked
		if _, err := exec.Command("git", "ls-files", "--error-unmatch", instanceDir).CombinedOutput(); err != nil {
			// Dir is not tracked
			if output, err := exec.Command("rm", "-rf", instanceDir).CombinedOutput(); err != nil {
				return newExecError(err, output, "rm", "-rf", instanceDir)
			}
		} else {
			// Dir is tracked
			if output, err := exec.Command("rm", "-f", instanceFile).CombinedOutput(); err != nil {
				return newExecError(err, output, "rm", "-rf", instanceDir)
			}
		}
	} else {
		// File is tracked
		args := []string{"reset", "HEAD", instanceFile}
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			return newExecError(err, output, "git", args...)
		}
		args = []string{"checkout", instanceFile}
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			return newExecError(err, output, "git", args...)
		}
	}
	return nil
}

func newExecError(err error, output []byte, cmd string, args ...string) error {
	debugExecOutput(output, cmd, args...)
	c := fmt.Sprintf("%s %s", cmd, strings.Join(args, " "))
	return errors.New(fmt.Sprintf("ERROR running `%s`:\n\t%s: %s", c, err.Error(), string(output)))
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

func indent(output []byte) string {
	return strings.Replace(string(output), "\n", "\n\t", -1)
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

func ConvertStartStopToSliceIndexes(start, stop, length int32) (int32, int32) {
	if stop > length {
		stop = length
	}
	if start <= int32(0) {
		start = 0
	} else {
		start = start - 1
	}
	return start, stop
}
