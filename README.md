# wsl-lan-bridge

CLI assistida para o **Windows** expor portas do **WSL 2** na rede local (LAN), sem precisar decorar `netsh`, `portproxy` nem regras de firewall.

**Manual completo (por quê, para quê, casos de uso):** [docs/MANUAL.md](docs/MANUAL.md)

## Por que usar?

No WSL 2 (modo NAT), o Linux tem um IP interno diferente do IP do Windows na Wi‑Fi/Ethernet. Outros dispositivos na rede — e às vezes apps no próprio Windows — não alcançam o serviço se você só “abrir a porta no Linux”.

Este utilitário:

- descobre a interface certa e o IPv4 de **`eth0`** no WSL (evita IPs de bridge Docker e falhas comuns do PowerShell);
- configura **portproxy** + **firewall** (ou só firewall no modo espelhado);
- testa a porta e guarda a config para um `refresh` depois de reiniciar o WSL.

## Casos de uso (visão rápida)

| Situação | O que você ganha |
|----------|------------------|
| API/site no WSL, teste no celular na mesma Wi‑Fi | URL `http://IP-do-Windows:porta` |
| Outro PC na LAN precisa do backend no desktop | Mesmo IP/porta estáveis do lado Windows |
| Docker Compose dentro do WSL | Encaminhamento sem receita manual de `netsh` |
| Após `wsl --shutdown` | `refresh` atualiza o IP interno automaticamente |

Mais detalhes e o que **não** fazer: [docs/MANUAL.md](docs/MANUAL.md).

## O que faz (tecnicamente)

1. Detecta distros WSL 2 e o modo de rede (`nat` ou `mirrored`).
2. Lista Wi‑Fi/Ethernet com gateway (ignora `vEthernet (WSL)` e bridges óbvias).
3. Lê o IPv4 de **`eth0`** no Linux.
4. **NAT:** portproxy Windows → WSL + firewall (perfil Privado, subnet local).
5. **Espelhado:** firewall (+ regra Hyper‑V se disponível).
6. Testa TCP no IP do Windows e salva a configuração.

## Requisitos

- Windows 10/11 com **WSL 2**
- Permissão de administrador (UAC) **somente** ao aplicar/remover regras
- Rede local marcada como **Privada** nas configurações do Windows

Não instala WSL, Docker nem a sua aplicação — só a ponte de rede.

## Uso

Baixe o `wsl-lan-bridge.exe` na [página de Releases](../../releases) ou compile (abaixo).

```text
wsl-lan-bridge              # assistente (perguntas numeradas)
wsl-lan-bridge apply --port 3000
wsl-lan-bridge status
wsl-lan-bridge refresh      # após wsl --shutdown / reboot
wsl-lan-bridge remove
```

Exemplo não interativo:

```text
wsl-lan-bridge apply --port 3000 --distro Ubuntu --listen 192.168.0.15 --mode nat
```

Configuração salva em `%LOCALAPPDATA%\wsl-lan-bridge\config.json`.

## Compilar

```sh
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o wsl-lan-bridge.exe ./cmd/wsl-lan-bridge
```

## Limitações

- Não abre portas no roteador nem na internet.
- O IP interno do WSL pode mudar após reinício: use `refresh`.
- Nos clientes, use o IPv4 do Wi‑Fi/Ethernet do Windows — não o IP `172.x` do WSL.

## Licença

MIT
