package wsl

import "testing"

func TestDecodeWSL_stripsNulls(t *testing.T) {
	in := []byte{'1', 0, '7', 0, '2', 0, '.', 0, '1', 0, '9', 0}
	got := decodeWSL(in)
	if got != "172.19" {
		t.Fatalf("got %q", got)
	}
}

func TestDecodeWSL_plain(t *testing.T) {
	got := decodeWSL([]byte("  172.19.107.160\n"))
	if got != "172.19.107.160" {
		t.Fatalf("got %q", got)
	}
}

func TestEth0Regex(t *testing.T) {
	line := "2: eth0    inet 172.19.107.160/20 brd 172.19.111.255 scope global eth0"
	m := ipv4InCIDR.FindStringSubmatch(line)
	if len(m) < 2 || m[1] != "172.19.107.160" {
		t.Fatalf("match=%v", m)
	}
}
