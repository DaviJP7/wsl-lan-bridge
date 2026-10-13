//go:build windows

package netwin

import (
	"bufio"
	"bytes"
	"net"
	"os/exec"
	"strings"
)

// defaultGateways maps interface name -> gateway IPv4 using route print.
func defaultGateways() (map[string]string, error) {
	cmd := exec.Command("route", "print", "-4")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	out = bytes.ReplaceAll(out, []byte{0}, nil)
	gatewaysByIP := map[string]string{} // interface IP -> gateway
	scanner := bufio.NewScanner(bytes.NewReader(out))
	inTable := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "Active Routes") || strings.Contains(line, "Rotas Ativas") {
			inTable = true
			continue
		}
		if !inTable {
			continue
		}
		if strings.HasPrefix(line, "=") || strings.HasPrefix(line, "Persistent") || strings.HasPrefix(line, "Persistentes") {
			if len(gatewaysByIP) > 0 && (strings.HasPrefix(line, "Persistent") || strings.HasPrefix(line, "Persistentes")) {
				break
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if fields[0] != "0.0.0.0" {
			continue
		}
		gw := fields[2]
		ifaceIP := fields[3]
		if net.ParseIP(gw) == nil || net.ParseIP(ifaceIP) == nil {
			continue
		}
		if gw == "On-link" || gw == "on-link" {
			continue
		}
		gatewaysByIP[ifaceIP] = gw
	}

	result := map[string]string{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return gatewaysByIP, nil
	}
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ip, ok := ipv4FromAddr(addr)
			if !ok {
				continue
			}
			if gw, found := gatewaysByIP[ip]; found {
				result[iface.Name] = gw
			}
		}
	}
	return result, nil
}
