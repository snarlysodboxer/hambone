// +build debug

package helpers

import (
	"log"
	"strings"
)

// TODO rename these
func Println(args ...interface{}) {
	log.Println(args...)
}

func Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func PrintExecOutput(output []byte, cmd string, args ...string) {
	log.Printf("Ran `%s %s` and got the following output:\n\t%s\n", cmd, strings.Join(args, " "), Indent(output))
}
