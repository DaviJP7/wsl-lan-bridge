//go:build !windows

package netwin

import "errors"

func FirewallUpsert(ruleName, listenIP string, port int) error {
	return errors.New("firewall só está disponível no Windows")
}

func FirewallRemove(ruleName string) error {
	return errors.New("firewall só está disponível no Windows")
}

func FirewallHyperVUpsert(ruleName string, port int) error {
	return errors.New("firewall Hyper-V só está disponível no Windows")
}

func FirewallHyperVRemove(ruleName string) error {
	return errors.New("firewall Hyper-V só está disponível no Windows")
}

func FirewallExists(ruleName string) (bool, error) {
	return false, errors.New("firewall só está disponível no Windows")
}
