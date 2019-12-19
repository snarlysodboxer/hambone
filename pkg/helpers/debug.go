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

// Debugln prints a debug line if debug logging is enabled
func Debugln(args ...interface{}) {
	logger.Println(args...)
}

// Debugf format prints debug info if debug logging is enabled
func Debugf(format string, args ...interface{}) {
	logger.Printf(format, args...)
}

// DebugExecOutput formats and prints exec.Command debug info if debug logging is enabled
func DebugExecOutput(output []byte, cmd string, args ...string) {
	logger.Printf("Ran `%s %s` and got the following output:\n\t%s\n", cmd, strings.Join(args, " "), Indent(output))
}
