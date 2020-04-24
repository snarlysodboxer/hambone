package git

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	pb "github.com/snarlysodboxer/hambone/generated"
	"github.com/snarlysodboxer/hambone/pkg/helpers"
	"github.com/snarlysodboxer/hambone/pkg/state"
	"github.com/snarlysodboxer/hambone/pkg/state/etcd"
	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	dialTimeout = 5 * time.Second
)

var (
	// TrackedUncommittedChangesError indicates a request's Instance has tracked but uncommitted changes in the working tree. This could indicate a bug, and likely needs fixed manually.
	TrackedUncommittedChangesError = status.Error(codes.FailedPrecondition, "tracked uncommitted changes")
	// UnTrackedUncommittedChangesError indicates a request's Instance has untracked and uncommitted changes in the working tree. This could indicate a bug, and likely needs fixed manually.
	UnTrackedUncommittedChangesError = status.Error(codes.FailedPrecondition, "untracked uncommitted changes")
	endpoints                        = []string{}
	etcdLocksGitKey                  = ""
)

// Engine fulfills the StateStore interface
type Engine struct {
	WorkingDir      string
	EndpointsString string
	EtcdLocksGitKey string
	Branch          string
}

// Init does setup
func (engine *Engine) Init() error {
	err := os.Chdir(engine.WorkingDir)
	if err != nil {
		return status.Errorf(codes.Unknown, "error changing directory: %s", err.Error())
	}
	// last branch may have been deleted, so start by checking out master
	// TODO consider not hard-coding master branch here
	output, err := exec.Command("git", "checkout", "master").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed for unknown reasons: %+v, %v", err, output)
	}
	err = gitPull()
	if err != nil {
		return err
	}
	output, err = exec.Command("git", "checkout", engine.Branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed for unknown reasons: %+v, %v", err, output)
	}

	etcdLocksGitKey = engine.EtcdLocksGitKey

	// parse and set endpoints
	endpoints = strings.Split(engine.EndpointsString, ",")

	return nil
}

// NewGetter returns an initialized Getter
func (engine *Engine) NewGetter(options *pb.GetOptions, list *pb.InstanceList, instancesDir string) state.Getter {
	return &gitGetter{options, list, instancesDir, engine.WorkingDir, nil}
}

// NewTemplatesGetter returns an initialized TemplatesGetter
func (engine *Engine) NewTemplatesGetter(list *pb.InstanceList, templatesDir string) state.TemplatesGetter {
	return &gitTemplatesGetter{list, templatesDir, engine.WorkingDir, nil}
}

// NewUpdater returns an initialized Updater
func (engine *Engine) NewUpdater(instance *pb.Instance, instancesDir string) state.Updater {
	instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, instance.Name)
	return &gitUpdater{instance, instanceDir, instanceFile, engine.WorkingDir, nil}
}

// NewDeleter returns an initialized Deleter
func (engine *Engine) NewDeleter(instance *pb.Instance, instancesDir string) state.Deleter {
	instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, instance.Name)
	return &gitDeleter{instance, instanceDir, instanceFile, engine.WorkingDir, nil}
}

type gitGetter struct {
	*pb.GetOptions
	*pb.InstanceList
	instancesDir string
	workingDir   string
	cleanupFuncs []func() error
}

// Run runs the get request
func (getter *gitGetter) Run() error {
	list := getter.InstanceList

	err := gitPull()
	if err != nil {
		return err
	}

	// list instances directory, sort
	files, err := ioutil.ReadDir(getter.instancesDir) // ReadDir sorts
	if err != nil {
		return status.Errorf(codes.Unknown, "error reading instancesDir: %s", err.Error())
	}
	if getter.GetOptions.GetName() != "" {
		for _, file := range files {
			if file.IsDir() {
				if file.Name() == getter.GetOptions.GetName() {
					err = loadInstance(getter.instancesDir, file.Name(), list, true)
					if err != nil {
						return err
					}
					break
				}
			}
		}
	} else {
		for _, file := range files {
			if file.IsDir() {
				err = loadInstance(getter.instancesDir, file.Name(), list, false)
				if err != nil {
					return err
				}
			}
		}

		// filter list to start and stop points in getOptions
		indexStart, indexStop := helpers.ConvertStartStopToSliceIndexes(getter.GetOptions.GetStart(), getter.GetOptions.GetStop(), int32(len(list.Instances)))
		if indexStop == 0 {
			list.Instances = list.Instances[indexStart:]
		} else {
			list.Instances = list.Instances[indexStart:indexStop]
		}
	}

	return nil
}

func (getter *gitGetter) RunCleanupFuncs() error {
	err := runCleanupFuncs(getter.cleanupFuncs)
	getter.cleanupFuncs = nil
	return err
}

type gitTemplatesGetter struct {
	*pb.InstanceList
	templatesDir string
	workingDir   string
	cleanupFuncs []func() error
}

// Run runs the get template request
func (getter *gitTemplatesGetter) Run() error {
	list := getter.InstanceList

	err := gitPull()
	if err != nil {
		return err
	}

	// list templates directory, sort
	files, err := ioutil.ReadDir(getter.templatesDir) // ReadDir sorts
	if err != nil {
		return status.Errorf(codes.Unknown, "error reading templatesDir: %s", err.Error())
	}
	for _, file := range files {
		if file.IsDir() {
			err = loadInstance(getter.templatesDir, file.Name(), list, false)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (getter *gitTemplatesGetter) RunCleanupFuncs() error {
	err := runCleanupFuncs(getter.cleanupFuncs)
	getter.cleanupFuncs = nil
	return err
}

type gitUpdater struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
	workingDir   string
	cleanupFuncs []func() error
}

// Init is expected to do any init related to the state store,
//   as well as write the kustomization.yaml file
func (updater *gitUpdater) Init() error {
	instanceFile := updater.instanceFile
	instanceDir := updater.instanceDir

	// take out a mutex lock if desired
	if etcdLocksGitKey != "" {
		clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
		if err != nil {
			return status.Errorf(codes.Unknown, "error getting etcd client: %s", err.Error())
		}
		cancel, session, mutex, err := etcd.GetSessionAndMutex(clientV3, etcdLocksGitKey, "updater")
		if err != nil {
			return status.Errorf(codes.Unknown, "error getting etcd session: %s", err.Error())
		}
		updater.cleanupFuncs = append(updater.cleanupFuncs, func() error {
			cancel()
			helpers.Debugln("Ran updater CancelFunc")
			return nil
		})
		updater.cleanupFuncs = append(updater.cleanupFuncs, func() error {
			if err = session.Close(); err != nil {
				return err
			}
			helpers.Debugln("Closed updater session")
			return nil
		})
		updater.cleanupFuncs = append(updater.cleanupFuncs, func() error {
			key := mutex.Key()
			if err = mutex.Unlock(context.TODO()); err != nil {
				return err
			}
			helpers.Debugf("Released updater lock for %s", key)
			return nil
		})
	}

	err := gitPull()
	if err != nil {
		return err
	}

	err = ensureCleanFile(instanceDir)
	if err != nil {
		return err
	}

	// compare oldInstance to existing if desired
	if updater.Instance.OldInstance != nil {
		if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
			return state.InstanceNoExistError
		}
		contents, err := ioutil.ReadFile(instanceFile)
		if err != nil {
			return status.Errorf(codes.Unknown, "error reading instanceFile: %s", err.Error())
		}
		if strings.TrimSpace(updater.Instance.OldInstance.KustomizationYaml) != strings.TrimSpace(string(contents)) {
			return state.OldInstanceDiffersError
		}
	}

	// mkdir
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return status.Errorf(codes.Unknown, "%s", err.Error())
	}

	// write <instancesDir>/<name>/kustomization.yaml
	if err := ioutil.WriteFile(instanceFile, []byte(updater.Instance.KustomizationYaml), 0644); err != nil {
		return status.Errorf(codes.Unknown, "%s", err.Error())
	}
	helpers.Debugf("Wrote `%s` with contents:\n\t%s\n", instanceFile, helpers.Indent([]byte(updater.Instance.KustomizationYaml)))

	for _, file := range updater.Instance.Files {
		// mkdir
		if err := os.MkdirAll(fmt.Sprintf("%s/%s", instanceDir, file.Directory), 0755); err != nil {
			return status.Errorf(codes.Unknown, "%s", err.Error())
		}

		// write <instancesDir>/<name>
		if err := ioutil.WriteFile(fmt.Sprintf("%s/%s", instanceDir, file.Name), []byte(file.Contents), 0644); err != nil {
			return status.Errorf(codes.Unknown, "%s", err.Error())
		}
		helpers.Debugf("Wrote `%s` with contents:\n\t%s\n", file.Name, helpers.Indent([]byte(file.Contents)))
	}

	return nil
}

// Cancel is expected to clean up any mess, and remove the kustomization.yaml file/dir
func (updater *gitUpdater) Cancel(err error) error {
	// TODO this needs to rollback
	return err
}

// Commit is expected to add/update the Instance in the state store
func (updater *gitUpdater) Commit(skipCommit bool) error {
	instanceFile := updater.instanceFile
	instanceDir := updater.instanceDir

	// check if there's anything to commit
	args := []string{`diff`, `--exit-code`, `HEAD`, updater.workingDir}
	output, err := exec.Command("git", args...).CombinedOutput()
	helpers.DebugExecOutput(output, "git", args...)
	if err != nil {
		// There are changes to commit

		// git add
		args = []string{"add", instanceDir}
		if _, err := rollbackCommand(instanceDir, instanceFile, "git", args...); err != nil {
			return status.Errorf(codes.Unknown, "git add error: %s", err)
		}

		if !skipCommit {
			// git commit
			args = []string{"commit", "-m", fmt.Sprintf(`Automate hambone apply for %s`, updater.Instance.Name)}
			if _, err := rollbackCommand(instanceDir, instanceFile, "git", args...); err != nil {
				return status.Errorf(codes.Unknown, "git commit error: %s", err)
			}

			// git push
			if _, err = rollbackCommand(instanceDir, instanceFile, "git", "push"); err != nil {
				return status.Errorf(codes.Unknown, "git push error: %s", err)
			}
		}
	}

	return nil
}

func (updater *gitUpdater) RunCleanupFuncs() error {
	err := runCleanupFuncs(updater.cleanupFuncs)
	updater.cleanupFuncs = nil
	return err
}

type gitDeleter struct {
	*pb.Instance
	instanceDir  string
	instanceFile string
	workingDir   string
	cleanupFuncs []func() error
}

// Init is expected to do any init related to the state store,
//   as well as write the kustomization.yaml file
func (deleter *gitDeleter) Init() error {
	instanceFile := deleter.instanceFile
	instanceDir := deleter.instanceDir

	// take out a mutex lock if desired
	if etcdLocksGitKey != "" {
		clientV3, err := clientv3.New(clientv3.Config{Endpoints: endpoints, DialTimeout: dialTimeout})
		if err != nil {
			return status.Errorf(codes.Unknown, "error getting etcd client: %s", err.Error())
		}
		cancel, session, mutex, err := etcd.GetSessionAndMutex(clientV3, etcdLocksGitKey, "deleter")
		if err != nil {
			return status.Errorf(codes.Unknown, "error getting etcd session: %s", err.Error())
		}
		deleter.cleanupFuncs = append(deleter.cleanupFuncs, func() error {
			cancel()
			helpers.Debugln("Ran deleter CancelFunc")
			return nil
		})
		deleter.cleanupFuncs = append(deleter.cleanupFuncs, func() error {
			if err = session.Close(); err != nil {
				return err
			}
			helpers.Debugln("Closed deleter session")
			return nil
		})
		deleter.cleanupFuncs = append(deleter.cleanupFuncs, func() error {
			key := mutex.Key()
			if err = mutex.Unlock(context.TODO()); err != nil {
				return err
			}
			helpers.Debugf("Released deleter lock for %s", key)
			return nil
		})
	}

	err := gitPull()
	if err != nil {
		return err
	}

	// ensure Instance exists
	if _, err = os.Stat(instanceFile); os.IsNotExist(err) {
		return status.Errorf(codes.NotFound, "Instance not found at `%s`", instanceFile)
	}

	err = ensureCleanFile(instanceDir)
	if err != nil {
		return err
	}

	// compare oldInstance to existing if desired
	if deleter.Instance.OldInstance != nil {
		if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
			return state.InstanceNoExistError
		}
		contents, err := ioutil.ReadFile(instanceFile)
		if err != nil {
			return status.Errorf(codes.Unknown, "error reading instanceFile: %s", err.Error())
		}
		if strings.TrimSpace(deleter.Instance.OldInstance.KustomizationYaml) != strings.TrimSpace(string(contents)) {
			return state.OldInstanceDiffersError
		}
	}

	return nil
}

// Cancel is expected to clean up any mess, and re-add the kustomization.yaml file/dir
func (deleter *gitDeleter) Cancel(err error) error {
	// TODO
	return nil
}

// Commit is expected to delete the Instance from the state store
func (deleter *gitDeleter) Commit() error {
	instanceDir := deleter.instanceDir

	// TODO consider the case where any of the following fail, but the objects have been deleted from k8s

	// git rm <instancesDir>/<name>/kustomization.yaml
	output, err := exec.Command("git", "rm", "-rf", instanceDir).CombinedOutput()
	helpers.DebugExecOutput(output, "git", "rm", "-rf", instanceDir)
	if err != nil {
		return status.Errorf(codes.Unknown, "git rm -rf error: %s", err)
	}

	// git commit
	args := []string{"commit", "-m", fmt.Sprintf(`Automate hambone delete for %s`, deleter.Instance.Name)}
	output, err = exec.Command("git", args...).CombinedOutput()
	helpers.DebugExecOutput(output, "git", args...)
	if err != nil {
		return status.Errorf(codes.Unknown, "git commit error: %s", err)
	}

	// TODO think about retries here and other places
	// git push
	output, err = exec.Command("git", "push").CombinedOutput()
	helpers.DebugExecOutput(output, "git", "push")
	if err != nil {
		return status.Errorf(codes.Unknown, "git push error: %s", err)
	}

	return nil
}

func (deleter *gitDeleter) RunCleanupFuncs() error {
	err := runCleanupFuncs(deleter.cleanupFuncs)
	deleter.cleanupFuncs = nil
	return err
}

func runCleanupFuncs(funcs []func() error) error {
	// reverse slice order
	for i := len(funcs)/2 - 1; i >= 0; i-- {
		opp := len(funcs) - 1 - i
		funcs[i], funcs[opp] = funcs[opp], funcs[i]
	}

	errorString := "Cleanup Errors: "
	isError := false
	for _, fn := range funcs {
		err := fn()
		if err != nil {
			isError = true
			name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
			errorString += fmt.Sprintf("\n%s: %s\n", name, err.Error())
		}
	}
	if isError {
		return status.Errorf(codes.Unknown, errorString)
	}

	return nil
}

func rollbackCommand(instanceDir, instanceFile, cmd string, args ...string) ([]byte, error) {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	helpers.DebugExecOutput(output, cmd, args...)
	if err != nil {
		return output, rollbackAndError(instanceDir, instanceFile, err)
	}
	return output, nil
}

func rollbackAndError(instanceDir, instanceFile string, err error) error {
	rollbackErr := rollback(instanceDir, instanceFile)
	if rollbackErr != nil {
		return fmt.Errorf("error rolling back!\n%s\n%s", helpers.Indent([]byte(rollbackErr.Error())), fmt.Sprintf("%s\n", err.Error()))
	}
	return err
}

func rollback(instanceDir, instanceFile string) error {
	if _, err := exec.Command("git", "ls-files", "--error-unmatch", instanceFile).CombinedOutput(); err != nil {
		// File is not tracked
		if _, err := exec.Command("git", "ls-files", "--error-unmatch", instanceDir).CombinedOutput(); err != nil {
			// Dir is not tracked
			if output, err := exec.Command("rm", "-rf", instanceDir).CombinedOutput(); err != nil {
				return helpers.NewExecError(err, output, "rm", "-rf", instanceDir)
			}
		} else {
			// Dir is tracked
			if output, err := exec.Command("rm", "-f", instanceFile).CombinedOutput(); err != nil {
				return helpers.NewExecError(err, output, "rm", "-rf", instanceDir)
			}
		}
	} else {
		// File is tracked
		args := []string{"reset", "HEAD", instanceFile}
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			return helpers.NewExecError(err, output, "git", args...)
		}
		args = []string{"checkout", instanceFile}
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			return helpers.NewExecError(err, output, "git", args...)
		}
	}
	return nil
}

func loadInstance(instancesDir, name string, list *pb.InstanceList, required bool) error {
	instanceDir, instanceFile := helpers.GetInstanceDirFile(instancesDir, name)
	if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
		if required {
			return fmt.Errorf("Found directory `%s/%s` but it does not contain a `%s` file", instancesDir, name, helpers.KustomizationFileName)
		}
		log.Printf("WARNING found directory `%s/%s` that does not contain a `%s` file, skipping\n", instancesDir, name, helpers.KustomizationFileName)
		return nil
	}

	contents, err := ioutil.ReadFile(instanceFile)
	if err != nil {
		return status.Errorf(codes.Unknown, "error reading %s: %s", instanceFile, err.Error())
	}
	instance := &pb.Instance{Name: name, KustomizationYaml: string(contents)}
	err = filepath.Walk(instanceDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return status.Errorf(codes.Unknown, "error walking dir %s: %s", path, err.Error())
			}
			// don't load directories as Instance Files
			if info.IsDir() {
				return nil
			}
			contents, err = ioutil.ReadFile(path)
			if err != nil {
				return status.Errorf(codes.Unknown, "error reading %s: %s", path, err.Error())
			}
			relativePath, err := filepath.Rel(instanceDir, path)
			if err != nil {
				return status.Errorf(codes.Unknown, "error getting relative path: %s", err.Error())
			}
			// skip the main kustomization.yaml file
			if relativePath == helpers.KustomizationFileName {
				return nil
			}
			pbFile := &pb.File{Name: relativePath, Directory: filepath.Dir(relativePath), Contents: string(contents)}
			instance.Files = append(instance.Files, pbFile)
			return nil
		})
	if err != nil {
		return status.Errorf(codes.Unknown, "error walking templatesDir: %s", err.Error())
	}
	list.Instances = append(list.Instances, instance)

	return nil
}

func gitPull() error {
	output, err := exec.Command("git", "pull").CombinedOutput()
	if err != nil {
		helpers.DebugExecOutput(output, "git", "pull")
		if strings.Contains(string(output), "Connection timed out") {
			return status.Errorf(codes.Unavailable, "git pull failed with a connection timeout: %s", err.Error())
		}
		return status.Errorf(codes.Unknown, "git pull failed for unknown reasons: %s", err.Error())
	}

	return nil
}

// ensure Instance dir is clean in git
func ensureCleanFile(instanceDir string) error {
	// check for tracked changes
	args := []string{`diff`, `--exit-code`, `--`, instanceDir}
	output, err := exec.Command("git", args...).CombinedOutput()
	helpers.DebugExecOutput(output, "git", args...)
	if err != nil {
		return TrackedUncommittedChangesError
	}

	// check for untracked changes
	test := fmt.Sprintf("git ls-files --exclude-standard --others %s", instanceDir)
	args = []string{`-c`, fmt.Sprintf("test -z $(%s)", test)}
	output, err = exec.Command("sh", args...).CombinedOutput()
	helpers.DebugExecOutput(output, "sh", args...)
	if err != nil {
		return UnTrackedUncommittedChangesError
	}

	return nil
}
