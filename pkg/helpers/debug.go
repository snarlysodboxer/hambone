// +build debug

package helpers

import (
	"log"
	"os"
	"strings"
)

var (
	logger = log.New(os.Stdout, "DEBUG: ", log.LstdFlags)
)

func Debugln(args ...interface{}) {
	logger.Println(args...)
}

func Debugf(format string, args ...interface{}) {
	logger.Printf(format, args...)
}

func DebugExecOutput(output []byte, cmd string, args ...string) {
	logger.Printf("Ran `%s %s` and got the following output:\n\t%s\n", cmd, strings.Join(args, " "), Indent(output))
}
