// +build !debug

package helpers

func Debugln(_ ...interface{}) {}

func Debugf(_ string, _ ...interface{}) {}

func DebugExecOutput(_ []byte, _ string, _ ...string) {}
