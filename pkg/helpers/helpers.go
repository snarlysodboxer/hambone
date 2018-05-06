package helpers

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	KustomizationFileName = "kustomization.yaml"
)

func Indent(output []byte) string {
	return strings.Replace(string(output), "\n", "\n\t", -1)
}

func NewExecError(err error, output []byte, cmd string, args ...string) error {
	DebugExecOutput(output, cmd, args...)
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

func GetInstanceDirFile(instancesDir, instanceName string) (string, string) {
	instanceDir := fmt.Sprintf(`%s/%s`, instancesDir, instanceName)
	instanceFile := fmt.Sprintf(`%s/%s`, instanceDir, KustomizationFileName)
	return instanceDir, instanceFile
}

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
