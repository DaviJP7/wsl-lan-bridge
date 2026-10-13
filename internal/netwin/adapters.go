package netwin

import (
	"fmt"
	"net"
	"strings"
)

// Adapter is a usable LAN interface with IPv4 and a default gateway.
type Adapter struct {
	Name    string
	IPv4    string
	Gateway string
}

func ListLANAdapters() ([]Adapter, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	gateways, err := defaultGateways()
	if err != nil {
		// Fall back to interfaces that look like LAN even without gateway map.
		gateways = map[string]string{}
	}
	var out []Adapter
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := iface.Name
		if skipAdapterName(name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, ok := ipv4FromAddr(addr)
			if !ok {
				continue
			}
			if skipPrivateNoise(ip) {
				continue
			}
			gw := gateways[name]
			if gw == "" {
				gw = gateways[strings.ToLower(name)]
			}
			// Prefer adapters that have a gateway; still list others as last resort.
			out = append(out, Adapter{Name: name, IPv4: ip, Gateway: gw})
			break
		}
	}
	withGW := make([]Adapter, 0, len(out))
	without := make([]Adapter, 0)
	for _, a := range out {
		if a.Gateway != "" {
			withGW = append(withGW, a)
		} else {
			without = append(without, a)
		}
	}
	if len(withGW) > 0 {
		return withGW, nil
	}
	if len(without) == 0 {
		return nil, fmt.Errorf("não encontrei Wi-Fi/Ethernet com IPv4; confira a conexão de rede")
	}
	return without, nil
}

func skipAdapterName(name string) bool {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "vethernet"):
		return true
	case strings.Contains(n, "wsl"):
		return true
	case strings.Contains(n, "docker"):
		return true
	case strings.Contains(n, "hyper-v"):
		return true
	case strings.Contains(n, "virtualbox"):
		return true
	case strings.Contains(n, "vmware"):
		return true
	case strings.Contains(n, "loopback"):
		return true
	case strings.HasPrefix(n, "br-"):
		return true
	case n == "docker0":
		return true
	default:
		return false
	}
}

func skipPrivateNoise(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		return true
	}
	// Common Hyper-V / VirtualBox host-only ranges without real LAN gateway.
	if strings.HasPrefix(ip, "172.19.") && strings.HasSuffix(ip, ".1") {
		// Often vEthernet (WSL) host side — already filtered by name, keep conservative.
	}
	return false
}

func ipv4FromAddr(addr net.Addr) (string, bool) {
	switch v := addr.(type) {
	case *net.IPNet:
		if v.IP.To4() == nil {
			return "", false
		}
		return v.IP.String(), true
	case *net.IPAddr:
		if v.IP.To4() == nil {
			return "", false
		}
		return v.IP.String(), true
	default:
		s := addr.String()
		host, _, err := net.ParseCIDR(s)
		if err == nil && host.To4() != nil {
			return host.String(), true
		}
		ip := net.ParseIP(s)
		if ip != nil && ip.To4() != nil {
			return ip.String(), true
		}
		return "", false
	}
}
