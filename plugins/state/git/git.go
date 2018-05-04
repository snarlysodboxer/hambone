package main

// Notes on this plugin
// The remaining notes and this plugin, while working is for those who can tollerate the occasional difficulties that arrise from the git server being down and from concurrency. (This would usually mean manually inspecting and cleaning in the repo, or deleting the pod it's running in and starting from the latest commit.) This code could well be improved but has been back-burnered in favor of working on the etcd plugin.

// * Setup lock plugin, with an in-memory option, (etcd later)
//     * With in-memory option, you can only run one replica of the app at a time
//     * With etcd option, you can run muliple replicas of the app

// ensure repo is clean and pulled at init time (push if needed)
// cluster-wide lock on any repo changes
// if update fails, reset repo, release lock, return error to caller
// if a committed push fails, retry and eventually, release lock, return error to caller, (shutdown, delete self-pod)?

// TODO periodically check that working tree is clean?

import (
	"errors"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/helpers"
	"github.com/snarlysodboxer/hambone/plugins/state"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

const (
	kustomizationFileName = "kustomization.yaml"
)

var (
	StateStore GitEngine
)

type GitEngine struct{}

func (engine *GitEngine) Init() error {
	// TODO Git pull here
	return nil
}

func (engine *GitEngine) NewUpdater(instance *pb.Instance, instancesDir string) state.Updater {
	instanceDir, instanceFile := getInstanceDirFile(instancesDir, instance.Name)
	return &GitUpdater{instance, instanceDir, instanceFile}
}

func (engine *GitEngine) NewDeleter(instance *pb.Instance, instancesDir string) state.Deleter {
	instanceDir, instanceFile := getInstanceDirFile(instancesDir, instance.Name)
	return &GitDeleter{instance, instanceDir, instanceFile}
}

func (engine *GitEngine) NewGetter(options *pb.GetOptions, list *pb.InstanceList, instancesDir string) state.Getter {
	return &GitGetter{options, list, instancesDir}
}

type GitGetter struct {
	*pb.GetOptions
	*pb.InstanceList
	instancesDir string
}

func (getter *GitGetter) Run() error {
	list := getter.InstanceList
	// git pull
	output, err := exec.Command("git", "pull").CombinedOutput()
	if err != nil {
		return newExecError(err, output, "git", "pull")
	}

	// list instances directory, sort
	files, err := ioutil.ReadDir(getter.instancesDir) // ReadDir sorts
	if err != nil {
		return err
	}
	// TODO DRY this out
	if getter.GetOptions.GetName() != "" {
		for _, file := range files {
			if file.IsDir() {
				if file.Name() == getter.GetOptions.GetName() {
					kFile := fmt.Sprintf("%s/%s/%s", getter.instancesDir, file.Name(), kustomizationFileName)
					if _, err := os.Stat(kFile); os.IsNotExist(err) {
						return errors.New(fmt.Sprintf("Found directory `%s/%s` but it does not contain a `%s` file", getter.instancesDir, file.Name(), kustomizationFileName))
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
				kFile := fmt.Sprintf("%s/%s/%s", getter.instancesDir, file.Name(), kustomizationFileName)
				if _, err := os.Stat(kFile); os.IsNotExist(err) {
					// TODO figure out how to warn here
					// debug("WARNING found directory `%s/%s` that does not contain a `%s` file, skipping", getter.instancesDir, file.Name(), kustomizationFileName)
					fmt.Println("WARNING found directory `%s/%s` that does not contain a `%s` file, skipping", getter.instancesDir, file.Name(), kustomizationFileName)
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
		indexStart, indexStop := ConvertStartStopToSliceIndexes(getter.GetOptions.GetStart(), getter.GetOptions.GetStop(), int32(len(list.Instances)))
		if indexStop == 0 {
			list.Instances = list.Instances[indexStart:]
		} else {
			list.Instances = list.Instances[indexStart:indexStop]
		}
	}

	return nil
}

type GitUpdater struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
}

// Init is expected to do any init related to the state store,
//   as well as write the kustomization.yaml file
func (updater *GitUpdater) Init() error {
	instanceFile := updater.instanceFile
	instanceDir := updater.instanceDir

	// TODO take out a mutex lock here

	// ensure instance file is clean in git
	// check for tracked changes
	args := []string{`diff`, `--exit-code`, `--`, instanceFile}
	output, err := exec.Command("git", args...).CombinedOutput()
	helpers.PrintExecOutput(output, "git", args...)
	if err != nil {
		return errors.New(fmt.Sprintf("There are tracked uncommitted changes for this Instance! This should not happen and could indicate a bug. Fix this manually:\n%s\n", helpers.Indent(output)))
	}
	// check for untracked changes
	test := fmt.Sprintf("git ls-files --exclude-standard --others %s", instanceFile)
	args = []string{`-c`, fmt.Sprintf("test -z $(%s)", test)}
	output, err = exec.Command("sh", args...).CombinedOutput()
	helpers.PrintExecOutput(output, "sh", args...)
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

	// write <instancesDir>/<name>/kustomization.yaml
	if err := ioutil.WriteFile(instanceFile, []byte(updater.Instance.KustomizationYaml), 0644); err != nil {
		return err
	}
	helpers.Printf("Wrote `%s` with contents:\n\t%s\n", instanceFile, helpers.Indent([]byte(updater.Instance.KustomizationYaml)))

	return nil
}

// Cancel is expected to clean up any mess, and remove the kustomization.yaml file/dir
func (updater *GitUpdater) Cancel(err error) error {
	// TODO this needs to rollback
	return err
}

// Commit is expected to add/update the Instance in the state store
func (updater *GitUpdater) Commit() error {
	instanceFile := updater.instanceFile
	instanceDir := updater.instanceDir

	// check if there's anything to commit
	args := []string{`diff`, `--exit-code`, instanceFile}
	output, err := exec.Command("git", args...).CombinedOutput()
	helpers.PrintExecOutput(output, "git", args...)
	test := fmt.Sprintf("git ls-files --exclude-standard --others %s", instanceFile)
	args = []string{`-c`, fmt.Sprintf("test -z $(%s)", test)}
	output, untrackedErr := exec.Command("sh", args...).CombinedOutput()
	helpers.PrintExecOutput(output, "sh", args...)
	if err != nil || untrackedErr != nil {
		// Changes to commit

		// git add
		args = []string{"add", instanceFile}
		if _, err := rollbackCommand(instanceDir, instanceFile, "git", args...); err != nil {
			return err
		}

		// git commit
		args = []string{"commit", "-m", fmt.Sprintf(`Automate hambone apply for %s`, updater.Instance.Name)}
		if _, err := rollbackCommand(instanceDir, instanceFile, "git", args...); err != nil {
			return err
		}

		// git push
		if _, err = rollbackCommand(instanceDir, instanceFile, "git", "push"); err != nil {
			return err
		}
	}

	return nil
}

type GitDeleter struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
}

// Init is expected to do any init related to the state store,
//   as well as write the kustomization.yaml file
func (deleter *GitDeleter) Init() error {
	instanceFile := deleter.instanceFile

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
	helpers.PrintExecOutput(output, "git", args...)
	if err != nil {
		return errors.New(fmt.Sprintf("There are tracked uncommitted changes for this Instance! This should not happen and could indicate a bug. Fix this manually:\n%s\n", helpers.Indent(output)))
	}
	// check for untracked changes
	test := fmt.Sprintf("git ls-files --exclude-standard --others %s", instanceFile)
	args = []string{`-c`, fmt.Sprintf("test -z $(%s)", test)}
	output, err = exec.Command("sh", args...).CombinedOutput()
	helpers.PrintExecOutput(output, "sh", args...)
	if err != nil {
		return errors.New("There are untracked uncommitted changes for this Instance! This should not happen and could indicate a bug. Fix this manually.")
	}

	return nil
}

// Cancel is expected to clean up any mess, and re-add the kustomization.yaml file/dir
func (deleter *GitDeleter) Cancel(err error) error {
	return nil
}

// Commit is expected to delete the Instance from the state store
func (deleter *GitDeleter) Commit() error {
	instanceFile := deleter.instanceFile

	// TODO consider the case where any of the following fail, but the objects have been deleted from k8s

	// git rm <instancesDir>/<name>/kustomization.yaml
	output, err := exec.Command("git", "rm", instanceFile).CombinedOutput()
	if err != nil {
		return newExecError(err, output, "git", "rm", instanceFile)
	}
	helpers.PrintExecOutput(output, "git", "rm", instanceFile)

	// git commit
	args := []string{"commit", "-m", fmt.Sprintf(`Automate hambone delete for %s`, deleter.Instance.Name)}
	output, err = exec.Command("git", args...).CombinedOutput()
	helpers.PrintExecOutput(output, "git", args...)
	if err != nil {
		return errors.New(fmt.Sprintf("ERROR with `git %s`, retry manually!\n%s%s", strings.Join(args, " "), err.Error(), helpers.Indent(output)))
	}

	// TODO think about retries here and other places
	// git push
	output, err = exec.Command("git", "push").CombinedOutput()
	helpers.PrintExecOutput(output, "git", "push")
	if err != nil {
		return errors.New(fmt.Sprintf("ERROR with `git push`, retry manually!\n%s%s", err.Error(), helpers.Indent(output)))
	}

	return nil
}

func rollbackCommand(instanceDir, instanceFile, cmd string, args ...string) ([]byte, error) {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	helpers.PrintExecOutput(output, cmd, args...)
	if err != nil {
		return output, rollbackAndError(instanceDir, instanceFile, err)
	}
	return output, nil
}

// TODO untangle rollback functions
func rollbackAndError(instanceDir, instanceFile string, err error) error {
	rollbackErr := rollback(instanceDir, instanceFile)
	if rollbackErr != nil {
		return errors.New(fmt.Sprintf("ERROR rolling back!\n%s\n%s\n", helpers.Indent([]byte(rollbackErr.Error())), err.Error()))
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
	helpers.PrintExecOutput(output, cmd, args...)
	c := fmt.Sprintf("%s %s", cmd, strings.Join(args, " "))
	return errors.New(fmt.Sprintf("ERROR running `%s`:\n\t%s: %s", c, err.Error(), string(output)))
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

func getInstanceDirFile(instancesDir, instanceName string) (string, string) {
	instanceDir := fmt.Sprintf(`%s/%s`, instancesDir, instanceName)
	instanceFile := fmt.Sprintf(`%s/%s`, instanceDir, kustomizationFileName)
	return instanceDir, instanceFile
}