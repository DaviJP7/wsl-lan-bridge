//go:build !windows

package netwin

import (
	"bufio"
	"net"
	"os"
	"strings"
)

func defaultGateways() (map[string]string, error) {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return map[string]string{}, nil
	}
	defer f.Close()
	result := map[string]string{}
	sc := bufio.NewScanner(f)
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		if fields[1] != "00000000" {
			continue
		}
		gwHex := fields[2]
		if len(gwHex) != 8 {
			continue
		}
		// Little-endian hex in /proc/net/route
		b := make([]byte, 4)
		for i := 0; i < 4; i++ {
			var v byte
			_, _ = parseHexByte(gwHex[i*2:i*2+2], &v)
			b[i] = v
		}
		ip := net.IPv4(b[0], b[1], b[2], b[3]).String()
		if ip == "0.0.0.0" {
			continue
		}
		result[fields[0]] = ip
	}
	return result, nil
}

func parseHexByte(s string, out *byte) (int, error) {
	var v byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		var n byte
		switch {
		case c >= '0' && c <= '9':
			n = c - '0'
		case c >= 'a' && c <= 'f':
			n = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			n = c - 'A' + 10
		default:
			return 0, os.ErrInvalid
		}
		v = v*16 + n
	}
	*out = v
	return 1, nil
}
