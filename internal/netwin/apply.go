package netwin

import (
	"fmt"
	"time"

	"github.com/david/wsl-lan-bridge/internal/config"
)

// ApplyParams describes a bridge configuration to apply.
type ApplyParams struct {
	Mode     string // nat | mirrored
	ListenIP string
	Port     int
	WSLIP    string // required for nat
	RuleName string
}

func (p ApplyParams) Normalize() ApplyParams {
	if p.RuleName == "" {
		p.RuleName = config.RuleName(p.Port)
	}
	return p
}

func Apply(p ApplyParams) error {
	p = p.Normalize()
	switch p.Mode {
	case "nat":
		if p.WSLIP == "" {
			return fmt.Errorf("modo NAT exige o IPv4 do WSL (eth0)")
		}
		if err := PortProxyAdd(p.ListenIP, p.Port, p.WSLIP, p.Port); err != nil {
			return err
		}
		if err := FirewallUpsert(p.RuleName, p.ListenIP, p.Port); err != nil {
			return fmt.Errorf("portproxy ok, mas firewall falhou: %w", err)
		}
	case "mirrored":
		if err := FirewallUpsert(p.RuleName, p.ListenIP, p.Port); err != nil {
			return err
		}
		_ = FirewallHyperVUpsert(p.RuleName+"-HV", p.Port)
	default:
		return fmt.Errorf("modo desconhecido: %s (use nat ou mirrored)", p.Mode)
	}
	return nil
}

func Remove(p ApplyParams) error {
	p = p.Normalize()
	var first error
	if p.Mode == "nat" || p.Mode == "" {
		if err := PortProxyDelete(p.ListenIP, p.Port); err != nil && first == nil {
			first = err
		}
	}
	if err := FirewallRemove(p.RuleName); err != nil && first == nil {
		first = err
	}
	_ = FirewallHyperVRemove(p.RuleName + "-HV")
	return first
}

func Verify(listenIP string, port int) error {
	return ProbeTCP(listenIP, port, 3*time.Second)
}

func StatusText() (string, error) {
	return PortProxyShow()
}
