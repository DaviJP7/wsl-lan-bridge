package app

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/david/wsl-lan-bridge/internal/config"
	"github.com/david/wsl-lan-bridge/internal/netwin"
	"github.com/david/wsl-lan-bridge/internal/wizard"
	"github.com/david/wsl-lan-bridge/internal/wsl"
)

func Run(args []string) int {
	if len(args) == 0 {
		return runWizard()
	}
	switch args[0] {
	case "help", "-h", "--help":
		printHelp()
		return 0
	case "apply":
		return cmdApply(args[1:])
	case "status":
		return cmdStatus()
	case "refresh":
		return cmdRefresh(hasElevated(args[1:]))
	case "remove":
		return cmdRemove(hasElevated(args[1:]))
	case "wizard":
		return runWizard()
	default:
		fmt.Fprintf(os.Stderr, "Comando desconhecido: %s\n\n", args[0])
		printHelp()
		return 2
	}
}

func printHelp() {
	fmt.Print(`wsl-lan-bridge — expõe portas do WSL 2 na rede local (Windows)

Uso:
  wsl-lan-bridge              assistente interativo
  wsl-lan-bridge wizard       idem
  wsl-lan-bridge apply --port 3000 [--distro Ubuntu] [--listen 192.168.0.15] [--mode nat|mirrored]
  wsl-lan-bridge status       mostra portproxy e última configuração salva
  wsl-lan-bridge refresh      reaplica a última configuração (útil após reiniciar o WSL)
  wsl-lan-bridge remove       remove portproxy/firewall da última configuração

A alteração de rede pede permissão de administrador (UAC).
Não instala WSL nem Docker; só configura encaminhamento e firewall.
`)
}

func hasElevated(args []string) bool {
	for _, a := range args {
		if a == "--elevated" {
			return true
		}
	}
	return false
}

func ensureAdmin(elevated bool, relaunchArgs []string) error {
	if netwin.IsAdmin() {
		return nil
	}
	if elevated {
		return fmt.Errorf("ainda sem privilégios de administrador")
	}
	fmt.Println("É necessário administrador para alterar portproxy/firewall.")
	fmt.Println("Uma janela do UAC deve abrir...")
	_, err := netwin.RelaunchElevated(relaunchArgs...)
	if err != nil {
		return err
	}
	// Parent exits after elevated child finishes successfully.
	os.Exit(0)
	return nil
}

func cmdStatus() int {
	ui := wizard.New(os.Stdin, os.Stdout)
	if saved, err := config.Load(); err == nil {
		ui.Println("Última configuração salva:")
		ui.Printf("  Distro:   %s\n", saved.Distro)
		ui.Printf("  Modo:     %s\n", saved.Mode)
		ui.Printf("  Escuta:   %s:%d\n", saved.ListenIP, saved.Port)
		if saved.ConnectIP != "" {
			ui.Printf("  WSL eth0: %s\n", saved.ConnectIP)
		}
		ui.Printf("  Firewall: %s\n", saved.FirewallRule)
		ui.Println()
	} else {
		ui.Println("Nenhuma configuração salva ainda.")
		ui.Println()
	}
	show, err := netwin.StatusText()
	if err != nil {
		ui.Printf("portproxy: %v\n", err)
		return 1
	}
	ui.Println("portproxy atual:")
	ui.Println(show)
	return 0
}

func cmdApply(args []string) int {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	port := fs.Int("port", 0, "porta TCP")
	distro := fs.String("distro", "", "nome da distro WSL")
	listen := fs.String("listen", "", "IPv4 Windows para escutar")
	mode := fs.String("mode", "", "nat ou mirrored")
	elevated := fs.Bool("elevated", false, "interno")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *port <= 0 || *port > 65535 {
		fmt.Fprintln(os.Stderr, "Informe --port (1-65535).")
		return 2
	}
	params, saved, err := resolveApply(*distro, *listen, *mode, *port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		return 1
	}
	if err := ensureAdmin(*elevated, append([]string{"apply"}, rebuildApplyArgs(params, saved.Distro)...)); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		return 1
	}
	if err := netwin.Apply(params); err != nil {
		fmt.Fprintf(os.Stderr, "Falha ao aplicar: %v\n", err)
		return 1
	}
	if err := config.Save(saved); err != nil {
		fmt.Fprintf(os.Stderr, "Aplicado, mas não salvei a config: %v\n", err)
	}
	printSuccess(params, saved)
	if err := netwin.Verify(params.ListenIP, params.Port); err != nil {
		fmt.Printf("Aviso: teste TCP em %s:%d falhou (%v).\n", params.ListenIP, params.Port, err)
		fmt.Println("Confira se o serviço está escutando no WSL nessa porta.")
		return 0
	}
	fmt.Println("Teste TCP: ok.")
	return 0
}

func rebuildApplyArgs(p netwin.ApplyParams, distro string) []string {
	args := []string{"--port", strconv.Itoa(p.Port), "--listen", p.ListenIP, "--mode", p.Mode}
	if distro != "" {
		args = append(args, "--distro", distro)
	}
	return args
}

func resolveApply(distroName, listenIP, mode string, port int) (netwin.ApplyParams, config.Saved, error) {
	wslPath, err := wsl.WslPath()
	if err != nil {
		return netwin.ApplyParams{}, config.Saved{}, err
	}
	distros, err := wsl.ListDistros(wslPath)
	if err != nil {
		return netwin.ApplyParams{}, config.Saved{}, err
	}
	var d wsl.Distro
	if distroName != "" {
		found := false
		for _, item := range distros {
			if strings.EqualFold(item.Name, distroName) {
				d = item
				found = true
				break
			}
		}
		if !found {
			return netwin.ApplyParams{}, config.Saved{}, fmt.Errorf("distro %q não encontrada", distroName)
		}
	} else {
		d = wsl.DefaultDistro(distros)
	}
	if err := wsl.EnsureWSL2(d); err != nil {
		return netwin.ApplyParams{}, config.Saved{}, err
	}
	if mode == "" {
		mode, err = wsl.NetworkingMode(wslPath, d.Name)
		if err != nil {
			return netwin.ApplyParams{}, config.Saved{}, err
		}
		if mode == "unknown" {
			mode = "nat"
		}
	}
	mode = strings.ToLower(mode)
	if mode != "nat" && mode != "mirrored" {
		return netwin.ApplyParams{}, config.Saved{}, fmt.Errorf("modo inválido: %s", mode)
	}
	if listenIP == "" {
		adapters, err := netwin.ListLANAdapters()
		if err != nil {
			return netwin.ApplyParams{}, config.Saved{}, err
		}
		listenIP = adapters[0].IPv4
	}
	var wslIP string
	if mode == "nat" {
		wslIP, err = wsl.Eth0IPv4(wslPath, d.Name)
		if err != nil {
			return netwin.ApplyParams{}, config.Saved{}, err
		}
	}
	params := netwin.ApplyParams{
		Mode:     mode,
		ListenIP: listenIP,
		Port:     port,
		WSLIP:    wslIP,
		RuleName: config.RuleName(port),
	}
	saved := config.Saved{
		Distro:       d.Name,
		Mode:         mode,
		ListenIP:     listenIP,
		Port:         port,
		ConnectIP:    wslIP,
		FirewallRule: params.RuleName,
	}
	return params, saved, nil
}

func cmdRefresh(elevated bool) int {
	saved, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Nada para atualizar: %v\nRode o assistente ou: wsl-lan-bridge apply --port ...\n", err)
		return 1
	}
	wslPath, err := wsl.WslPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		return 1
	}
	params := netwin.ApplyParams{
		Mode:     saved.Mode,
		ListenIP: saved.ListenIP,
		Port:     saved.Port,
		RuleName: saved.FirewallRule,
	}
	if saved.Mode == "nat" {
		ip, err := wsl.Eth0IPv4(wslPath, saved.Distro)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao ler eth0: %v\n", err)
			return 1
		}
		params.WSLIP = ip
		saved.ConnectIP = ip
	}
	if err := ensureAdmin(elevated, []string{"refresh"}); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		return 1
	}
	if err := netwin.Apply(params); err != nil {
		fmt.Fprintf(os.Stderr, "Falha: %v\n", err)
		return 1
	}
	_ = config.Save(saved)
	printSuccess(params, saved)
	if err := netwin.Verify(params.ListenIP, params.Port); err != nil {
		fmt.Printf("Aviso: teste TCP falhou (%v).\n", err)
		return 0
	}
	fmt.Println("Teste TCP: ok.")
	return 0
}

func cmdRemove(elevated bool) int {
	saved, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Nada para remover: %v\n", err)
		return 1
	}
	if err := ensureAdmin(elevated, []string{"remove"}); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		return 1
	}
	params := netwin.ApplyParams{
		Mode:     saved.Mode,
		ListenIP: saved.ListenIP,
		Port:     saved.Port,
		RuleName: saved.FirewallRule,
	}
	if err := netwin.Remove(params); err != nil {
		fmt.Fprintf(os.Stderr, "Falha parcial ao remover: %v\n", err)
	} else {
		fmt.Println("Regras removidas.")
	}
	_ = config.RemoveFile()
	return 0
}

func printSuccess(p netwin.ApplyParams, s config.Saved) {
	fmt.Println()
	fmt.Println("Pronto.")
	fmt.Printf("  Use nos outros computadores: %s  porta %d\n", p.ListenIP, p.Port)
	if p.Mode == "nat" {
		fmt.Printf("  Encaminhamento: %s:%d -> %s:%d (WSL eth0)\n", p.ListenIP, p.Port, p.WSLIP, p.Port)
		fmt.Println("  Depois de reiniciar o WSL, rode: wsl-lan-bridge refresh")
	} else {
		fmt.Println("  Modo espelhado: sem portproxy; firewall liberado.")
	}
	fmt.Printf("  Distro: %s\n", s.Distro)
	fmt.Println("  Remover depois: wsl-lan-bridge remove")
	fmt.Println()
}

func runWizard() int {
	ui := wizard.New(os.Stdin, os.Stdout)
	ui.Println("=== wsl-lan-bridge — assistente de rede WSL ↔ Windows ===")
	ui.Println("Expõe uma porta do Linux (WSL) no IP do Wi-Fi/Ethernet do Windows.")
	ui.Println()

	wslPath, err := wsl.WslPath()
	if err != nil {
		ui.Printf("Erro: %v\n", err)
		ui.Pause("")
		return 1
	}
	distros, err := wsl.ListDistros(wslPath)
	if err != nil {
		ui.Printf("Erro: %v\n", err)
		ui.Pause("")
		return 1
	}
	opts := make([]string, len(distros))
	defIdx := 0
	for i, d := range distros {
		mark := ""
		if d.Default {
			mark = " (padrão)"
			defIdx = i
		}
		state := "parada"
		if d.Running {
			state = "em execução"
		}
		opts[i] = fmt.Sprintf("%s — WSL%d, %s%s", d.Name, d.Version, state, mark)
	}
	idx, err := ui.Choose("Escolha a distribuição WSL:", opts)
	if err != nil {
		ui.Printf("Erro: %v\n", err)
		return 1
	}
	if idx < 0 {
		idx = defIdx
	}
	d := distros[idx]
	if err := wsl.EnsureWSL2(d); err != nil {
		ui.Printf("Erro: %v\n", err)
		ui.Pause("")
		return 1
	}

	mode, _ := wsl.NetworkingMode(wslPath, d.Name)
	if mode == "unknown" {
		midx, err := ui.Choose("Não detectei o modo de rede do WSL. Qual você usa?", []string{
			"NAT (padrão do WSL 2)",
			"Espelhado (mirrored)",
		})
		if err != nil {
			return 1
		}
		if midx == 1 {
			mode = "mirrored"
		} else {
			mode = "nat"
		}
	} else {
		ui.Printf("Modo de rede WSL detectado: %s\n", mode)
	}

	adapters, err := netwin.ListLANAdapters()
	if err != nil {
		ui.Printf("Erro: %v\n", err)
		ui.Pause("")
		return 1
	}
	aopts := make([]string, len(adapters))
	for i, a := range adapters {
		gw := a.Gateway
		if gw == "" {
			gw = "(sem gateway detectado)"
		}
		aopts[i] = fmt.Sprintf("%s — %s  gateway %s", a.Name, a.IPv4, gw)
	}
	aidx, err := ui.Choose("Escolha a conexão Wi-Fi/Ethernet da LAN (não use vEthernet WSL):", aopts)
	if err != nil {
		return 1
	}
	adapter := adapters[aidx]

	port, err := ui.AskInt("Porta TCP no WSL/Windows", 3000, 1, 65535)
	if err != nil {
		return 1
	}

	var wslIP string
	if mode == "nat" {
		wslIP, err = wsl.Eth0IPv4(wslPath, d.Name)
		if err != nil {
			ui.Printf("Erro ao ler IP eth0 do WSL: %v\n", err)
			ui.Pause("")
			return 1
		}
		ui.Printf("IP do WSL (eth0): %s\n", wslIP)
	}

	ui.Println()
	ui.Println("Resumo:")
	ui.Printf("  Distro:  %s\n", d.Name)
	ui.Printf("  Modo:    %s\n", mode)
	ui.Printf("  Windows: %s:%d (%s)\n", adapter.IPv4, port, adapter.Name)
	if mode == "nat" {
		ui.Printf("  WSL:     %s:%d\n", wslIP, port)
	}
	ok, err := ui.Confirm("Aplicar agora?", true)
	if err != nil || !ok {
		ui.Println("Cancelado.")
		return 0
	}

	params := netwin.ApplyParams{
		Mode:     mode,
		ListenIP: adapter.IPv4,
		Port:     port,
		WSLIP:    wslIP,
		RuleName: config.RuleName(port),
	}
	saved := config.Saved{
		Distro:       d.Name,
		Mode:         mode,
		ListenIP:     adapter.IPv4,
		Port:         port,
		ConnectIP:    wslIP,
		FirewallRule: params.RuleName,
	}

	if !netwin.IsAdmin() {
		relaunch := append([]string{"apply"}, rebuildApplyArgs(params, d.Name)...)
		ui.Println("Reabrindo com permissão de administrador (UAC) para aplicar...")
		if _, err := netwin.RelaunchElevated(relaunch...); err != nil {
			ui.Printf("Erro: %v\n", err)
			ui.Pause("")
			return 1
		}
		return 0
	}

	if err := netwin.Apply(params); err != nil {
		ui.Printf("Falha: %v\n", err)
		ui.Pause("")
		return 1
	}
	if err := config.Save(saved); err != nil {
		ui.Printf("Aviso: não salvei config (%v)\n", err)
	}
	printSuccess(params, saved)
	if err := netwin.Verify(params.ListenIP, params.Port); err != nil {
		ui.Printf("Aviso: teste TCP em %s:%d falhou (%v).\n", params.ListenIP, params.Port, err)
		ui.Println("Se o serviço ainda não estiver rodando no WSL, isso é esperado.")
	} else {
		ui.Println("Teste TCP: ok — a porta responde neste IP do Windows.")
	}
	ui.Pause("")
	return 0
}
