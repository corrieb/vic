// +build !linux

package daemon

func secretsSupported() bool {
	return false
}
