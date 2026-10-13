//go:build !windows

package netwin

import "errors"

func PortProxyAdd(listenIP string, listenPort int, connectIP string, connectPort int) error {
	return errors.New("portproxy só está disponível no Windows")
}

func PortProxyDelete(listenIP string, listenPort int) error {
	return errors.New("portproxy só está disponível no Windows")
}

func PortProxyShow() (string, error) {
	return "", errors.New("portproxy só está disponível no Windows")
}
