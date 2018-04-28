package instances

import (
	"errors"
	"fmt"
	pb "github.com/snarlysodboxer/hambone/generated"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// TODO periodically check that working tree is clean?

func (server *instancesServer) apply(instance *pb.Instance) (*pb.Instance, error) {
	instanceDir := fmt.Sprintf(`%s/%s`, server.instancesDir, instance.Name)
	instanceFile := fmt.Sprintf(`%s/kustomization.yaml`, instanceDir)

	// ensure namePrefix in yaml matches pb.Instance.Name
	err := namePrefixMatches(instance)
	if err != nil {
		return instance, err
	}

	// ensure instance file is clean in git
	// check for tracked changes
	args := []string{`diff`, `--exit-code`, instanceFile}
	output, err := exec.Command("git", args...).CombinedOutput()
	debugExecOutput(output, "git", args...)
	if err != nil {
		return instance, errors.New(fmt.Sprintf("There are tracked uncommitted changes for this Instance! This should not happen and could indicate a bug. Fix this manually:\n%s\n", indent(output)))
	}
	// check for untracked changes
	test := fmt.Sprintf("git ls-files --exclude-standard --others %s", instanceFile)
	args = []string{`-c`, fmt.Sprintf("test -z $(%s)", test)}
	output, err = exec.Command("sh", args...).CombinedOutput()
	debugExecOutput(output, "sh", args...)
	if err != nil {
		return instance, errors.New("There are untracked uncommitted changes for this Instance! This should not happen and could indicate a bug. Fix this manually.")
	}

	// git pull
	output, err = exec.Command("git", "pull").CombinedOutput()
	debugExecOutput(output, "git", "pull")
	if err != nil {
		return instance, newExecError(err, output, "git", []string{"pull"})
	}

	// mkdir
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return instance, err
	}

	// write <instancesDir>/<name>/kustomization.yml
	if err := ioutil.WriteFile(instanceFile, []byte(instance.KustomizationYaml), 0644); err != nil {
		return instance, err
	}
	debugf("Wrote `%s` with contents:\n%s\n", instanceFile, indent([]byte(instance.KustomizationYaml)))

	// kustomization build | kubctl apply
	// TODO use exec pipes for this for better error handling
	args = []string{"-c", fmt.Sprintf(`kustomize build %s | kubectl apply -f -`, instanceDir)}
	if _, err = rollbackCommand(instanceFile, "sh", args...); err != nil {
		return instance, err
	}

	// check if there's anything to commit
	args = []string{`diff`, `--exit-code`, instanceFile}
	output, err = exec.Command("git", args...).CombinedOutput()
	debugExecOutput(output, "git", args...)
	if err == nil {
		// Nothing to commit
		return instance, nil
	}

	// git add
	args = []string{"add", instanceFile}
	if _, err := rollbackCommand(instanceFile, "git", args...); err != nil {
		return instance, err
	}

	// git commit
	args = []string{"commit", "-m", `Automated commit by hambone`}
	if _, err := rollbackCommand(instanceFile, "git", args...); err != nil {
		return instance, err
	}

	// git push
	if _, err = rollbackCommand(instanceFile, "git", "push"); err != nil {
		return instance, err
	}

	return instance, nil
}

func rollbackCommand(instanceFile, cmd string, args ...string) ([]byte, error) {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	debugExecOutput(output, cmd, args...)
	if err != nil {
		if rollbackErr := rollback(instanceFile); rollbackErr != nil {
			return output, rollbackErr
		}
		return output, newExecError(err, output, cmd, args)
	}
	return output, nil
}

func rollback(instanceFile string) error {
	args := []string{"ls-files", "--error-unmatch", instanceFile}
	if _, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		// File is not tracked
		args = []string{"-f", instanceFile}
		if output, err := exec.Command("rm", args...).CombinedOutput(); err != nil {
			return newExecError(err, output, "rm", args)
		}
	} else {
		// File is tracked
		args = []string{"reset", "HEAD", instanceFile}
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			return newExecError(err, output, "git", args)
		}
		args = []string{"checkout", instanceFile}
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			return newExecError(err, output, "git", args)
		}
	}
	return nil
}

func newExecError(err error, output []byte, cmd string, args []string) error {
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
