//go:build !debug

package build

func IsDebug() bool {
	return false
}
