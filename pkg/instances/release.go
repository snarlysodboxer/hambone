// +build !debug

package instances

func debug(args ...interface{}) {}

func debugf(fmt string, args ...interface{}) {}

func debugExecOutput(output []byte, cmd string, args ...interface{}) {}

func indent(output []byte) string {}
