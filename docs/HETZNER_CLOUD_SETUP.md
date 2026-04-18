# Hetzner Cloud — notas de setup (ChatBot / WA SaaS)

Referência rápida para decisões na consola ao criar o servidor. Alinhado ao playbook: `docs/saas-whatsapp-playbook.md` (secção Deploy).

## SSH key

- Par gerado no PC (não no repositório): `%USERPROFILE%\.ssh\hetzner_wa_saas_ed25519` (privada) e `.pub` (pública).
- Na Hetzner: colar **só** o conteúdo do `.pub` no campo **SSH key**; **Name** sugerido: `wa-saas-chatbot-pc` (ou outro identificador do teu equipamento).
- Ligação: `ssh -i %USERPROFILE%\.ssh\hetzner_wa_saas_ed25519 root@<IP>`.

## Volumes (passo “Volumes” / modal “Create Volume”)

- **O que é:** disco em bloco extra (SSD), **além** do disco incluído no tipo de servidor (ex.: CX31 com ~80 GB no próprio VPS).
- **Precisas disto agora?** Na maioria dos casos **não**. Docker/Coolify/Postgres com dados em volumes no disco do servidor costumam caber no disco do plano.
- **Se não criares Volume:** fecha o modal com **Cancel** ou ignora **+ Create Volume** e segue o wizard — **não há custo** de Volume.
- **Aviso importante (Hetzner):** se clicares **Create & Buy now** no Volume, **passam a facturar o Volume mesmo que não completes** a criação do servidor. Só criar Volume se quiseres mesmo esse recurso.
- **Nome:** se criares um volume, usa um nome descritivo (ex.: `nbg1-chatbot-data`); a **localização** tem de ser a **mesma** do servidor (ex.: Nuremberg / `nbg1`).
- **Filesystem:** **EXT4** é adequado por defeito; **XFS** é alternativa comum para certos padrões de I/O — para este projeto, EXT4 é suficiente na maior parte dos cenários.

## Firewalls (criação do servidor)

- **Não és obrigado a fazer nada aqui** para concluir o assistente.
- Se ainda **não tens** firewalls no projeto, não há nada para escolher — podes **avançar**.
- **Depois:** em **Networking → Firewalls** podes criar regras (ex.: TCP 22 só do teu IP, 80/443 públicos) e **associar ao servidor**. Útil como camada extra; Cloudflare + UFW no VPS também entram na história.

## Backups (Hetzner)

- **Opcional.** Se o checkbox **Backups** ficar **desligado**, **não** pagas o suplemento (~20 % do preço do servidor; o disco de **Volumes** não entra nestes backups).
- **Ligado:** cópias diárias automáticas do disco do servidor para restauração rápida.
- **Recomendação inicial:** podes **deixar desligado** para poupar custo e ativar no painel quando quiseres essa comodidade; mantém na mesma estratégia de backups da app (DB dumps, etc.) conforme o playbook.

## Placement groups

- **Para que serve:** agrupar **vários** servidores para a Hetzner os colocar em **hosts físicos diferentes** (tipo **Spread**), reduzindo o risco de caírem todos ao mesmo tempo se um nó de hardware falhar.
- **Um só VPS (o teu caso):** **não precisas** disto. Clica **Cancel** no modal ou não abras **+ Create placement group** e **segue** o assistente.
- **Quando faz sentido:** quando tiveres **2+ servidores** na mesma região e quiseres essa garantia de anti-afinidade ao nível do hardware.

## Labels

- **Opcional.** São pares **chave=valor** (ou só chave) para organizar recursos na consola, relatórios ou automação — **não** alteram rede nem desempenho.
- **Podes deixar o campo vazio** e criar o servidor sem problema.
- **Se quiseres marcar o recurso:** exemplos válidos (um por linha, conforme o formato que o assistente pedir), e.g. `env=production`, `app=wa-chatbot`, `project=chatbot` — respeita as regras da Hetzner (tamanho, caracteres permitidos).

## Cloud config (cloud-init)

- **Opcional.** É o **cloud-init**: YAML/shell a correr no **primeiro arranque** (pacotes, utilizadores, `runcmd`, etc.), até ~32 KiB.
- **Para o fluxo típico (SSH → atualizar sistema → Coolify / Docker):** **deixa vazio.** Configuras depois à mão ou com o instalador oficial do Coolify; evita duplicar lógica e erros num snippet colado na consola.
- **Quando preencher:** só se tiveres um **ficheiro cloud-init testado** (reprodutível, versionado no repo) — fora desse caso, vazio.

## Resumo prático neste ecrã

1. **Sem necessidade de disco extra:** não adicionar Volume → continuar o assistente.
2. **Com disco extra:** só depois de ter a certeza do tamanho e da localização; aceitar que o Volume fica na conta e é cobrado.
3. **Firewalls:** nada a selecionar se ainda não criaste um → seguir em frente (configurar firewall depois se quiseres).
4. **Backups:** deixar desmarcado é válido (menos custo); marcar só se quiseres backups geridos pela Hetzner.
5. **Placement groups:** ignorar com um único servidor; só relevante com vários servidores e requisitos de HA no hardware.
6. **Labels:** opcional; vazio é perfeito. Usa só se quiseres etiquetas na consola ou para futura organização.
7. **Cloud config:** opcional; vazio para começar por SSH e configurar depois (Coolify/Docker).
