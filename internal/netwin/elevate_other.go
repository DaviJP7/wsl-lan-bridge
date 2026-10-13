//go:build !windows

package netwin

import "errors"

func IsAdmin() bool { return false }

func RelaunchElevated(extraArgs ...string) (int, error) {
	return 1, errors.New("elevação UAC só existe no Windows; compile e rode o .exe no Windows")
}
