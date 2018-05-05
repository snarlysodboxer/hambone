package helpers

import (
	"errors"
	"fmt"
	"strings"
)

const (
	KustomizationFileName = "kustomization.yaml"
)

func Indent(output []byte) string {
	return strings.Replace(string(output), "\n", "\n\t", -1)
}

func NewExecError(err error, output []byte, cmd string, args ...string) error {
	PrintExecOutput(output, cmd, args...)
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
