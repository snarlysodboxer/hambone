// +build debug

package instances

import (
	"log"
	"strings"
)

func debug(args ...interface{}) {
	log.Println(args...)
}

func debugf(fmt string, args ...interface{}) {
	log.Printf(fmt, args...)
}

func debugExecOutput(output []byte, cmd string, args ...string) {
	log.Printf("Ran `%s %s` and got the following output:\n\t%s\n", cmd, strings.Join(args, " "), indent(output))
}

func indent(output []byte) string {
	return strings.Replace(string(output), "\n", "\n\t", -1)
}
