//go:build windows

package netwin

import (
	"fmt"
	"os/exec"
	"strings"
)

func FirewallUpsert(ruleName, listenIP string, port int) error {
	_ = FirewallRemove(ruleName)
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
New-NetFirewallRule -Name '%s' -DisplayName 'WSL LAN Bridge %d' -Direction Inbound -Action Allow -Protocol TCP -LocalPort %d -LocalAddress '%s' -Profile Private -RemoteAddress LocalSubnet | Out-Null
`, escapePS(ruleName), port, port, escapePS(listenIP))
	return runPowerShell(script)
}

func FirewallRemove(ruleName string) error {
	script := fmt.Sprintf(`
Get-NetFirewallRule -Name '%s' -ErrorAction SilentlyContinue | Remove-NetFirewallRule
`, escapePS(ruleName))
	return runPowerShell(script)
}

func FirewallHyperVUpsert(ruleName string, port int) error {
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
if (-not (Get-Command New-NetFirewallHyperVRule -ErrorAction SilentlyContinue)) {
  throw 'New-NetFirewallHyperVRule indisponível; atualize Windows/WSL para o modo espelhado.'
}
Get-NetFirewallHyperVRule -Name '%s' -ErrorAction SilentlyContinue | Remove-NetFirewallHyperVRule
New-NetFirewallHyperVRule -Name '%s' -DisplayName 'WSL LAN Bridge Hyper-V %d' -Direction Inbound -VMCreatorId '{40E0AC32-46A5-438A-A0B2-2B479E8F2E90}' -Protocol TCP -LocalPorts %d | Out-Null
`, escapePS(ruleName), escapePS(ruleName), port, port)
	return runPowerShell(script)
}

func FirewallHyperVRemove(ruleName string) error {
	script := fmt.Sprintf(`
if (Get-Command Remove-NetFirewallHyperVRule -ErrorAction SilentlyContinue) {
  Get-NetFirewallHyperVRule -Name '%s' -ErrorAction SilentlyContinue | Remove-NetFirewallHyperVRule
}
`, escapePS(ruleName))
	return runPowerShell(script)
}

func FirewallExists(ruleName string) (bool, error) {
	script := fmt.Sprintf(`
$r = Get-NetFirewallRule -Name '%s' -ErrorAction SilentlyContinue
if ($r) { 'yes' } else { 'no' }
`, escapePS(ruleName))
	out, err := runPowerShellOutput(script)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "yes", nil
}

func runPowerShell(script string) error {
	_, err := runPowerShellOutput(script)
	return err
}

func runPowerShellOutput(script string) (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	out, err := cmd.CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		if s == "" {
			s = err.Error()
		}
		return s, fmt.Errorf("PowerShell: %s", s)
	}
	return s, nil
}

func escapePS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
