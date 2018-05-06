// +build debug

package helpers

import (
	"log"
	"strings"
)

func Debugln(args ...interface{}) {
	log.Println(args...)
}

func Debugf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func DebugExecOutput(output []byte, cmd string, args ...string) {
	log.Printf("Ran `%s %s` and got the following output:\n\t%s\n", cmd, strings.Join(args, " "), Indent(output))
}
