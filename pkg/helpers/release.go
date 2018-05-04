// +build !debug

package helpers

func Println(_ ...interface{}) {}

func Printf(_ string, _ ...interface{}) {}

func PrintExecOutput(_ []byte, _ string, _ ...string) {}
