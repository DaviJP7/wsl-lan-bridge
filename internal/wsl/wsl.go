package wsl

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var ipv4InCIDR = regexp.MustCompile(`inet\s+(\d{1,3}(?:\.\d{1,3}){3})/\d+`)

// Distro describes a WSL distribution.
type Distro struct {
	Name    string
	Running bool
	Version int
	Default bool
}

func WslPath() (string, error) {
	if runtime.GOOS != "windows" {
		// Allow development from Linux hosts that expose wsl.exe via interop.
		if p, err := exec.LookPath("wsl.exe"); err == nil {
			return p, nil
		}
		if p, err := exec.LookPath("wsl"); err == nil {
			return p, nil
		}
		return "", errors.New("este programa configura a rede no Windows; rode o .exe no Windows com WSL 2")
	}
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}
	p := filepath.Join(systemRoot, "System32", "wsl.exe")
	if _, err := os.Stat(p); err != nil {
		if lp, lookErr := exec.LookPath("wsl.exe"); lookErr == nil {
			return lp, nil
		}
		return "", errors.New("wsl.exe não encontrado; instale o WSL 2 primeiro")
	}
	return p, nil
}

func run(wslPath string, args ...string) (string, error) {
	cmd := exec.Command(wslPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := decodeWSL(stdout.Bytes())
	if err != nil {
		msg := strings.TrimSpace(decodeWSL(stderr.Bytes()))
		if msg == "" {
			msg = strings.TrimSpace(out)
		}
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("%s", msg)
	}
	return out, nil
}

// decodeWSL strips UTF-16 NUL padding that PowerShell/WSL often inject into captures.
func decodeWSL(b []byte) string {
	if len(b) >= 2 && b[1] == 0 {
		var cleaned []byte
		for i := 0; i < len(b); i++ {
			if b[i] != 0 {
				cleaned = append(cleaned, b[i])
			}
		}
		b = cleaned
	} else {
		b = bytes.ReplaceAll(b, []byte{0}, nil)
	}
	return strings.TrimSpace(string(b))
}

func ListDistros(wslPath string) ([]Distro, error) {
	out, err := run(wslPath, "--list", "--verbose")
	if err != nil {
		return nil, fmt.Errorf("não foi possível listar distros WSL: %w", err)
	}
	lines := strings.Split(out, "\n")
	var distros []Distro
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(strings.ToUpper(line), "NAME") {
			continue
		}
		defaultMark := strings.HasPrefix(line, "*")
		line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		name := fields[0]
		if strings.EqualFold(name, "docker-desktop") || strings.EqualFold(name, "docker-desktop-data") {
			continue
		}
		state := strings.ToLower(fields[1])
		version := 2
		fmt.Sscanf(fields[2], "%d", &version)
		distros = append(distros, Distro{
			Name:    name,
			Running: state == "running",
			Version: version,
			Default: defaultMark,
		})
	}
	if len(distros) == 0 {
		return nil, errors.New("nenhuma distribuição WSL encontrada (além do Docker Desktop)")
	}
	return distros, nil
}

func DefaultDistro(distros []Distro) Distro {
	for _, d := range distros {
		if d.Default {
			return d
		}
	}
	return distros[0]
}

func Exec(wslPath, distro string, command ...string) (string, error) {
	args := []string{"-d", distro, "--exec"}
	args = append(args, command...)
	return run(wslPath, args...)
}

func NetworkingMode(wslPath, distro string) (string, error) {
	out, err := Exec(wslPath, distro, "sh", "-c", "if command -v wslinfo >/dev/null 2>&1; then wslinfo --networking-mode; else echo unknown; fi")
	if err != nil {
		return "unknown", nil
	}
	mode := strings.ToLower(strings.TrimSpace(out))
	switch mode {
	case "nat", "mirrored":
		return mode, nil
	default:
		return "unknown", nil
	}
}

func Eth0IPv4(wslPath, distro string) (string, error) {
	out, err := Exec(wslPath, distro, "ip", "-4", "-o", "addr", "show", "dev", "eth0")
	if err != nil {
		// Fallback: first global address on eth0 via addr show.
		out, err = Exec(wslPath, distro, "sh", "-c", "ip -4 addr show eth0 | awk '/inet /{print $2; exit}'")
		if err != nil {
			return "", fmt.Errorf("não encontrei IPv4 em eth0 na distro %s: %w", distro, err)
		}
		ip, _, cutErr := net.ParseCIDR(strings.TrimSpace(out))
		if cutErr == nil && ip != nil {
			return ip.String(), nil
		}
		ip2 := net.ParseIP(strings.TrimSpace(out))
		if ip2 == nil || ip2.To4() == nil {
			return "", fmt.Errorf("saída inesperada de eth0: %q", out)
		}
		return ip2.String(), nil
	}
	m := ipv4InCIDR.FindStringSubmatch(out)
	if len(m) < 2 {
		return "", fmt.Errorf("não encontrei IPv4 em eth0 (saída: %q)", out)
	}
	ip := net.ParseIP(m[1])
	if ip == nil || ip.To4() == nil {
		return "", fmt.Errorf("IPv4 inválido em eth0: %s", m[1])
	}
	return ip.String(), nil
}

func EnsureWSL2(d Distro) error {
	if d.Version != 2 {
		return fmt.Errorf("a distro %s está em WSL %d; converta para WSL 2 (wsl --set-version %s 2)", d.Name, d.Version, d.Name)
	}
	return nil
}
