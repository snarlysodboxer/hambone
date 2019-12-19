package helpers

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	// KustomizationFileName defines the name of the kustomization.yaml file
	KustomizationFileName = "kustomization.yaml"
)

// Indent indents a string for readability
func Indent(output []byte) string {
	return strings.Replace(string(output), "\n", "\n\t", -1)
}

// NewExecError returns an error formatted for exec.Command output
func NewExecError(err error, output []byte, cmd string, args ...string) error {
	DebugExecOutput(output, cmd, args...)
	c := fmt.Sprintf("%s %s", cmd, strings.Join(args, " "))
	return fmt.Errorf("ERROR running `%s`:\n\t%s: %s", c, err.Error(), string(output))
}

// ConvertStartStopToSliceIndexes converts start and stop numbers into slice indexes
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

// GetInstanceDirFile returns the instanceDir and instanceFile names
func GetInstanceDirFile(instancesDir, instanceName string) (string, string) {
	instanceDir := fmt.Sprintf(`%s/%s`, instancesDir, instanceName)
	instanceFile := fmt.Sprintf(`%s/%s`, instanceDir, KustomizationFileName)
	return instanceDir, instanceFile
}

// MkdirFile ensures the directory and file exist
func MkdirFile(filePath, contents string) error {
	// mkdir
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		log.Println(err)
		return err
	}

	// write file
	if err := ioutil.WriteFile(filePath, []byte(contents), 0644); err != nil {
		log.Println(err)
		return err
	}
	Debugf("Wrote `%s` with contents:\n\t%s\n", filePath, Indent([]byte(contents)))

	return nil
}

// IsEmpty determines if a path is empty or not
func IsEmpty(path string) (bool, error) {
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
