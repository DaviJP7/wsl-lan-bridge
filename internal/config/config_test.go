package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuleName(t *testing.T) {
	if RuleName(3000) != "WslLanBridge-3000" {
		t.Fatal(RuleName(3000))
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	// On Windows UserConfigDir uses APPDATA; force via HOME layout for Unix test.
	if err := os.Setenv("HOME", dir); err != nil {
		t.Fatal(err)
	}
	// UserConfigDir on Unix is $XDG_CONFIG_HOME or $HOME/.config
	_ = os.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "cfg"))

	s := Saved{
		Distro:    "Ubuntu",
		Mode:      "nat",
		ListenIP:  "192.168.0.12",
		Port:      3000,
		ConnectIP: "172.19.107.160",
	}
	if err := Save(s); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Distro != s.Distro || got.ListenIP != s.ListenIP || got.Port != s.Port {
		t.Fatalf("%+v", got)
	}
	if got.FirewallRule != "WslLanBridge-3000" {
		t.Fatal(got.FirewallRule)
	}
}
