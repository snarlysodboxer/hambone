// +build debug

package helpers

import (
	"fmt"
	"strings"
)

func Println(args ...interface{}) {
	fmt.Println(args...)
}

func Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func PrintExecOutput(output []byte, cmd string, args ...string) {
	fmt.Printf("Ran `%s %s` and got the following output:\n\t%s\n", cmd, strings.Join(args, " "), Indent(output))
}
