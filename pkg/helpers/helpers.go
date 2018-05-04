package helpers

import (
	"strings"
)

func Indent(output []byte) string {
	return strings.Replace(string(output), "\n", "\n\t", -1)
}
