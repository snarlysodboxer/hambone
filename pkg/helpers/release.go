// +build !debug

package helpers

// Debugln prints a debug line if debug logging is enabled
func Debugln(_ ...interface{}) {}

// Debugf format prints debug info if debug logging is enabled
func Debugf(_ string, _ ...interface{}) {}

// DebugExecOutput formats and prints exec.Command debug info if debug logging is enabled
func DebugExecOutput(_ []byte, _ string, _ ...string) {}
