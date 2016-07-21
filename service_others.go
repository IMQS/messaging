// +build !windows

package messaging

func RunAsService(handler func()) bool {
	return false
}
