# Manual — wsl-lan-bridge

Guia em português: **por que** a ferramenta existe, **para que** serve e **quando** usar.
Não substitui a documentação oficial do WSL da Microsoft; complementa o dia a dia de quem desenvolve no Windows com Linux dentro do WSL 2.

## Em uma frase

O `wsl-lan-bridge` é um assistente de console no Windows que expõe uma porta do WSL 2 no IP da sua rede Wi‑Fi/Ethernet, sem você precisar montar à mão `netsh portproxy`, regras de firewall e o IP interno correto do Linux.

## Por que isso é necessário?

No WSL 2 o Linux roda em uma máquina virtual leve. No modo de rede **NAT** (o padrão), o WSL ganha um IPv4 próprio (em geral na faixa `172.x`), separado do IP do Windows na LAN.

Consequências comuns:

1. **Outro computador na mesma rede** (notebook, celular, tablet, Raspberry Pi) **não alcança** `localhost` nem o IP interno do WSL. Ele só conhece o IP do Windows (ex.: `192.168.0.15`).
2. Mesmo no **próprio Windows**, vários apps e testes falham se você apontar para o IP errado (`vEthernet (WSL)`, bridge do Docker, etc.).
3. A documentação e fóruns pedem comandos longos (`netsh`, PowerShell como admin, firewall). É fácil:
   - pegar o IP errado (`hostname -I` lista várias interfaces, inclusive bridges Docker);
   - quebrar a captura no PowerShell (saída do WSL com caracteres nulos);
   - esquecer o firewall ou o perfil de rede **Privada**;
   - perder o encaminhamento depois que o IP do WSL muda ao reiniciar.

O modo **espelhado** (*mirrored*) muda o cenário (menos portproxy), mas ainda exige liberar o firewall corretamente — e nem todo mundo usa esse modo.

## Para que serve?

Serve para **publicar na LAN uma porta TCP que já está escutando dentro do WSL**, usando o IPv4 “visível” do Windows.

Fluxo típico:

```text
[outro device / app Windows]
        |
        |  TCP → 192.168.x.x:PORTA
        v
[Windows]  --portproxy / firewall-->  [WSL eth0:PORTA]
                                          |
                                     seu serviço (Node, Go, Docker, etc.)
```

A ferramenta **não**:

- instala WSL, Docker ou o seu projeto;
- abre portas no roteador ou na internet;
- cria VPN, túnel Cloudflare, ngrok, etc.;
- configura DNS ou HTTPS por conta própria.

Ela só cuida da **ponte Windows ↔ WSL** na rede local.

## Casos de uso

### 1. API ou site rodando no WSL, testado no celular

Você sobe `npm run dev` ou um servidor HTTP na porta `3000` dentro do Ubuntu WSL. No celular (mesma Wi‑Fi), abre `http://IP-DO-WINDOWS:3000`. Sem o encaminhamento, a conexão falha ou trava.

### 2. App em outro PC da casa/escritório

Notebook Linux ou Mac na mesma rede precisa falar com um backend que só existe no desktop Windows+WSL (porta `8080`, `5173`, `5432` publicado, etc.).

### 3. Docker Compose dentro do WSL

Containers publicam portas no Linux. Do ponto de vista da LAN, ainda é o WSL quem escuta; o Windows precisa encaminhar (NAT) ou liberar (mirrored).

### 4. Preview de frontend + backend separados

Vite/Webpack na `5173` e API na `8080`: você pode rodar o assistente duas vezes (ou `apply` duas vezes) para cada porta, ou focar na porta que os outros dispositivos realmente precisam.

### 5. Evitar “receita de PowerShell” em equipe

Em time ou open source, em vez de colar blocos frágeis de `netsh` no README do projeto, aponta-se para este utilitário: perguntas numeradas, teste TCP e `refresh` depois de reiniciar o WSL.

### 6. Reaplicar depois de `wsl --shutdown`

O IP de `eth0` muda com frequência. `wsl-lan-bridge refresh` relê o IP atual e recria o portproxy com a última porta/interface salvas.

## Quando *não* usar

| Situação | Alternativa |
|----------|-------------|
| Só você no mesmo Windows, e `localhost` / `127.0.0.1` já funciona | Não precisa desta ferramenta |
| Precisa de acesso pela internet | Túnel (Cloudflare Tunnel, Tailscale, etc.) ou hospedagem |
| Quer expor o WSL como máquina na LAN com IP próprio | Avaliar modos de rede avançados do WSL / bridge; fora do escopo atual |
| Distro em WSL 1 | Converta para WSL 2 |

## Conceitos rápidos

| Termo | Significado |
|-------|-------------|
| **IP do Windows** | IPv4 do Wi‑Fi ou Ethernet (o que o `ipconfig` mostra na conexão ativa). É o endereço que os outros usam. |
| **IP do WSL (`eth0`)** | IPv4 interno da VM Linux. Usado só no encaminhamento NAT; em geral **não** informe esse IP aos clientes remotos. |
| **portproxy** | Recurso do Windows que escuta em um IP:porta e encaminha para outro IP:porta (aqui, o WSL). |
| **Perfil Privado** | Na rede doméstica/trabalho, o Windows deve tratar a conexão como privada para as regras de firewall locais fazerem sentido. |

## Como usar (resumo)

1. No WSL, confirme que o serviço já responde (ex.: `curl http://127.0.0.1:3000`).
2. No Windows, execute `wsl-lan-bridge.exe` (assistente) ou `apply --port 3000`.
3. Aceite o UAC quando pedir administrador.
4. Anote o IP mostrado (IP do Windows) e use-o nos outros dispositivos.
5. Após reiniciar o WSL: `wsl-lan-bridge refresh`.
6. Para desfazer: `wsl-lan-bridge remove`.

Detalhes de comandos e compilação: [README](../README.md).

## Problemas frequentes

**O teste TCP falha, mas o portproxy aparece**  
O encaminhamento pode estar certo e o serviço no WSL parado ou em outra porta. Confira no Linux primeiro.

**Funciona no Windows e não no celular**  
Confira se ambos estão na mesma Wi‑Fi (não “convidados”), se a rede Windows está **Privada**, e se o celular usa o IP do Windows — não o `172.x` do WSL.

**Depois de reiniciar parou**  
Rode `refresh`. O IP interno do WSL provavelmente mudou.

**Várias interfaces listadas**  
Escolha Wi‑Fi ou Ethernet **com gateway** para o roteador. Evite adaptadores virtuais.

## Privacidade e escopo do projeto

Este repositório é uma ferramenta genérica de rede para WSL 2. Não inclui, recomenda nem documenta produtos comerciais específicos de terceiros. Use com a porta e o serviço que forem do seu fluxo de desenvolvimento.
