package wizard

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type UI struct {
	In  *bufio.Reader
	Out io.Writer
}

func New(in io.Reader, out io.Writer) *UI {
	return &UI{In: bufio.NewReader(in), Out: out}
}

func (u *UI) Println(a ...any) {
	fmt.Fprintln(u.Out, a...)
}

func (u *UI) Printf(format string, a ...any) {
	fmt.Fprintf(u.Out, format, a...)
}

func (u *UI) Pause(msg string) {
	if msg == "" {
		msg = "Pressione Enter para continuar..."
	}
	u.Printf("%s", msg)
	_, _ = u.In.ReadString('\n')
}

func (u *UI) AskString(prompt, def string) (string, error) {
	if def != "" {
		u.Printf("%s [%s]: ", prompt, def)
	} else {
		u.Printf("%s: ", prompt)
	}
	line, err := u.In.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
}

func (u *UI) AskInt(prompt string, def, min, max int) (int, error) {
	for {
		defStr := ""
		if def != 0 {
			defStr = strconv.Itoa(def)
		}
		s, err := u.AskString(prompt, defStr)
		if err != nil {
			return 0, err
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < min || n > max {
			u.Printf("Digite um número entre %d e %d.\n", min, max)
			continue
		}
		return n, nil
	}
}

func (u *UI) Choose(prompt string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("nenhuma opção disponível")
	}
	u.Println(prompt)
	for i, o := range options {
		u.Printf("  %d. %s\n", i+1, o)
	}
	n, err := u.AskInt("Digite o número", 1, 1, len(options))
	if err != nil {
		return -1, err
	}
	return n - 1, nil
}

func (u *UI) Confirm(prompt string, defYes bool) (bool, error) {
	hint := "s/N"
	if defYes {
		hint = "S/n"
	}
	s, err := u.AskString(prompt+" ("+hint+")", "")
	if err != nil {
		return false, err
	}
	if s == "" {
		return defYes, nil
	}
	s = strings.ToLower(s)
	return s == "s" || s == "sim" || s == "y" || s == "yes", nil
}
