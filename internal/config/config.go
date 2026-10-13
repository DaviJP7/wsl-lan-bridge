package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	AppName      = "wsl-lan-bridge"
	FirewallName = "WslLanBridge"
)

// Saved is the last successful apply, used by refresh/remove.
type Saved struct {
	Distro      string `json:"distro"`
	Mode        string `json:"mode"` // nat | mirrored
	ListenIP    string `json:"listen_ip"`
	Port        int    `json:"port"`
	ConnectIP   string `json:"connect_ip,omitempty"`
	FirewallRule string `json:"firewall_rule"`
}

func Dir() (string, error) {
	// Prefer %LOCALAPPDATA% on Windows (machine-local, not Roaming).
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, AppName), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppName), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (Saved, error) {
	path, err := Path()
	if err != nil {
		return Saved{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Saved{}, err
	}
	var s Saved
	if err := json.Unmarshal(raw, &s); err != nil {
		return Saved{}, err
	}
	if s.Port <= 0 || s.ListenIP == "" || s.Distro == "" {
		return Saved{}, errors.New("configuração salva incompleta")
	}
	if s.FirewallRule == "" {
		s.FirewallRule = RuleName(s.Port)
	}
	return s, nil
}

func Save(s Saved) error {
	if s.FirewallRule == "" {
		s.FirewallRule = RuleName(s.Port)
	}
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path, err := Path()
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func RemoveFile() error {
	path, err := Path()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func RuleName(port int) string {
	return FirewallName + "-" + itoa(port)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
