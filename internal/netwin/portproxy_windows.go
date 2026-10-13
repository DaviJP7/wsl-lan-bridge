//go:build windows

package netwin

import (
	"fmt"
	"os/exec"
	"strings"
)

func PortProxyAdd(listenIP string, listenPort int, connectIP string, connectPort int) error {
	_ = PortProxyDelete(listenIP, listenPort)
	cmd := exec.Command("netsh", "interface", "portproxy", "add", "v4tov4",
		fmt.Sprintf("listenaddress=%s", listenIP),
		fmt.Sprintf("listenport=%d", listenPort),
		fmt.Sprintf("connectaddress=%s", connectIP),
		fmt.Sprintf("connectport=%d", connectPort),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh portproxy add: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func PortProxyDelete(listenIP string, listenPort int) error {
	cmd := exec.Command("netsh", "interface", "portproxy", "delete", "v4tov4",
		fmt.Sprintf("listenaddress=%s", listenIP),
		fmt.Sprintf("listenport=%d", listenPort),
	)
	_, _ = cmd.CombinedOutput()
	return nil
}

func PortProxyShow() (string, error) {
	cmd := exec.Command("netsh", "interface", "portproxy", "show", "v4tov4")
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
