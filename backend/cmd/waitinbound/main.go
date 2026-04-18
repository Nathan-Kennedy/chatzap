// waitinbound: espera na BD por uma mensagem inbound (teste manual webhook → inbox).
//
//	Uso (com Postgres a correr e API a receber webhooks):
//	  cd backend && go run ./cmd/waitinbound -phone 5569993378283 -text Cuiudu
//	Diagnóstico (sem esperar): webhooks com re-parse, cronologia in/out, histórico por conversa:
//	  go run ./cmd/waitinbound -diag
//	  go run ./cmd/waitinbound -diag -diag-threads 8 -diag-thread-msgs 40
//	Reprocessar webhooks já gravados (dir=event) → inbox:
//	  go run ./cmd/waitinbound -reconcile -instance cuiudo_loja
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
	"wa-saas/backend/internal/service"
)

func main() {
	phone := flag.String("phone", "", "número com país, ex. 5569993378283 ou 69993378283")
	text := flag.String("text", "Cuiudu", "trecho do texto da mensagem recebida (contém, ignora maiúsculas)")
	timeout := flag.Duration("timeout", 3*time.Minute, "tempo máximo a esperar")
	interval := flag.Duration("interval", 2*time.Second, "intervalo entre consultas")
	diag := flag.Bool("diag", false, "mostra últimos webhooks e mensagens inbound e sai")
	diagWebhooks := flag.Int("diag-webhooks", 25, "com -diag: quantos webhook_messages mostrar")
	diagTimeline := flag.Int("diag-timeline", 40, "com -diag: linhas na cronologia geral (in+out)")
	diagThreads := flag.Int("diag-threads", 5, "com -diag: quantas conversas recentes com histórico")
	diagThreadMsgs := flag.Int("diag-thread-msgs", 25, "com -diag: mensagens por conversa no histórico")
	reconcile := flag.Bool("reconcile", false, "reprocessa webhook_messages e grava inbound em falta (use com -instance)")
	reconcileInstance := flag.String("instance", "", "evolution_instance_name (ex. cuiudo_loja); se vazio e só existir 1 instância na BD, usa essa")
	reconcileLimit := flag.Int("reconcile-limit", 500, "máximo de linhas webhook_messages a reprocessar")
	flag.Parse()

	for _, p := range []string{".env", "../backend/.env", "backend/.env"} {
		_ = godotenv.Load(p)
	}
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "defina DATABASE_URL (ex. no backend/.env)")
		os.Exit(2)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "postgres:", err)
		os.Exit(2)
	}

	if *reconcile {
		runReconcile(db, *reconcileInstance, *reconcileLimit)
		return
	}

	if *diag {
		runDiag(db, diagOpts{
			textSnippet: *text,
			webhooks:    *diagWebhooks,
			timeline:    *diagTimeline,
			threads:     *diagThreads,
			threadMsgs:  *diagThreadMsgs,
		})
		return
	}

	if strings.TrimSpace(*phone) == "" {
		fmt.Fprintln(os.Stderr, "obrigatório: -phone (ou usa -diag)")
		flag.Usage()
		os.Exit(2)
	}

	jid := service.NormalizeContactJID(*phone)
	if jid == "" {
		fmt.Fprintln(os.Stderr, "número/JID inválido:", *phone)
		os.Exit(2)
	}
	digits := digitsOnly(strings.Split(jid, "@")[0])
	likeDigits := digits + "@%"

	textLower := strings.ToLower(strings.TrimSpace(*text))

	fmt.Printf("A procurar inbound com %q em conversa JID=%q ou LIKE %q (timeout %s)…\n", *text, jid, likeDigits, *timeout)
	fmt.Println("Envia a mensagem pelo WhatsApp agora. Ctrl+C para cancelar.")
	deadline := time.Now().Add(*timeout)
	n := 0

	for time.Now().Before(deadline) {
		tx := db.Model(&model.Message{}).
			Joins("INNER JOIN conversations ON conversations.id = messages.conversation_id").
			Where("messages.direction = ?", "inbound").
			Where("(conversations.contact_j_id = ? OR conversations.contact_j_id LIKE ?)", jid, likeDigits).
			Order("messages.created_at DESC")

		var rows []model.Message
		if err := tx.Limit(50).Find(&rows).Error; err != nil {
			fmt.Fprintln(os.Stderr, "query:", err)
			os.Exit(2)
		}
		for i := range rows {
			if strings.Contains(strings.ToLower(rows[i].Body), textLower) {
				msg := rows[i]
				fmt.Println("OK — mensagem recebida e gravada na inbox.")
				fmt.Printf("  id=%s  created_at=%s\n  body=%q\n", msg.ID, msg.CreatedAt.UTC().Format(time.RFC3339), msg.Body)
				os.Exit(0)
			}
		}

		n++
		if n%10 == 0 {
			fmt.Printf("… ainda nada (%s restantes) — corre com -diag para ver se o webhook chegou\n", time.Until(deadline).Round(time.Second))
		}
		time.Sleep(*interval)
	}

	fmt.Println("TIMEOUT — não apareceu mensagem inbound com esse texto para esse contacto.")
	fmt.Println("  Corre: go run ./cmd/waitinbound -diag")
	fmt.Println("  Confirma: API a correr, Instâncias → sincronizar webhook, Evolution a alcançar host.docker.internal:<porta>.")
	os.Exit(1)
}

type diagOpts struct {
	textSnippet string
	webhooks    int
	timeline    int
	threads     int
	threadMsgs  int
}

func runReconcile(db *gorm.DB, instanceName string, limit int) {
	name := strings.TrimSpace(instanceName)
	if name == "" {
		var list []model.WhatsAppInstance
		if err := db.Find(&list).Error; err != nil {
			fmt.Fprintln(os.Stderr, "lista instâncias:", err)
			os.Exit(2)
		}
		if len(list) == 0 {
			fmt.Fprintln(os.Stderr, "não há whatsapp_instances na BD — regista uma instância no painel primeiro")
			os.Exit(2)
		}
		if len(list) > 1 {
			fmt.Fprintln(os.Stderr, "há várias instâncias: passa -instance com evolution_instance_name (coluna na BD)")
			for _, in := range list {
				fmt.Fprintf(os.Stderr, "  - %q  id=%s\n", in.EvolutionInstanceName, in.ID)
			}
			os.Exit(2)
		}
		name = list[0].EvolutionInstanceName
		fmt.Printf("(usando a única instância na BD: %q)\n", name)
	}

	var inst model.WhatsAppInstance
	if err := db.Where("LOWER(TRIM(evolution_instance_name)) = ?", strings.ToLower(strings.TrimSpace(name))).First(&inst).Error; err != nil {
		fmt.Fprintf(os.Stderr, "instância evolution_instance_name=%q não encontrada em whatsapp_instances (nome tem de bater com o path do webhook)\n", name)
		os.Exit(2)
	}

	log := zap.NewNop()
	n, scanned, err := service.ReconcileWebhooksToInbox(db, log, nil, inst.WorkspaceID, inst, "", limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reconcile:", err)
		os.Exit(2)
	}
	fmt.Printf("OK — webhook_messages analisados: %d  |  novas mensagens inbound gravadas: %d\n", scanned, n)
	if scanned > 0 && n == 0 {
		fmt.Println("Nada novo: ou o parse ainda falha nesses eventos, ou já estavam gravadas (dedupe por key.id), ou não há raw_payload.")
	}
	fmt.Println("Corre de novo: go run ./cmd/waitinbound -diag")
}

func truncateRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || s == "" {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-3]) + "..."
}

func summarizeWebhookReparse(raw []byte) string {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return "(sem raw_payload)"
	}
	var payload service.EvolutionWebhookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Sprintf("JSON inválido: %v", err)
	}
	data := service.NormalizeWebhookData(payload.Data)
	inbound, ok := service.ParseInboundFromEvolution(payload.Event, data)
	if !ok {
		if service.IsInboundMessageEvent(payload.Event) {
			if prev := service.WebhookMessageTextPreview(payload.Event, data); prev != "" {
				return fmt.Sprintf("parse incompleto — texto=%q (sem JID estável); com a API atualizada deve gravar na inbox", truncateRunes(prev, 100))
			}
			return "evento de mensagem mas parse atual não extrai texto/JID (tipo não suportado, fromMe, ou payload vazio)"
		}
		return "evento que não é mensagem parseável neste parser"
	}
	txt := truncateRunes(inbound.Text, 100)
	if inbound.FromMe {
		return fmt.Sprintf("re-parse: TU enviaste (fromMe) texto=%q", txt)
	}
	return fmt.Sprintf("re-parse: de %s texto=%q", inbound.From, txt)
}

func runDiag(db *gorm.DB, o diagOpts) {
	nw := o.webhooks
	if nw <= 0 {
		nw = 25
	}
	nt := o.timeline
	if nt <= 0 {
		nt = 40
	}
	nth := o.threads
	if nth <= 0 {
		nth = 5
	}
	nm := o.threadMsgs
	if nm <= 0 {
		nm = 25
	}

	fmt.Printf("=== Últimos %d webhook_messages (gravado na BD + re-parse do JSON) ===\n", nw)
	fmt.Println("(re-parse usa o código atual — útil para ver texto mesmo em linhas antigas gravadas como dir=event)")
	fmt.Println()
	var wms []model.WebhookMessage
	if err := db.Order("created_at DESC").Limit(nw).Find(&wms).Error; err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(wms) == 0 {
		fmt.Println("(vazio — nenhum POST de webhook gravado; Evolution não está a bater na API ou auth falhou)")
	} else {
		for _, w := range wms {
			body := w.Body
			if len(body) > 100 {
				body = body[:97] + "..."
			}
			fmt.Printf("%s  event=%s  inst=%s\n", w.CreatedAt.UTC().Format(time.RFC3339), w.Event, w.InstanceID)
			fmt.Printf("  BD: dir=%s  remote_jid=%q  body=%q\n", w.Direction, w.RemoteJID, body)
			fmt.Printf("  %s\n\n", summarizeWebhookReparse(w.RawPayload))
		}
	}

	fmt.Printf("=== Cronologia recente (recebidas + enviadas, últimas %d) ===\n", nt)
	type tlRow struct {
		CreatedAt time.Time
		Direction string
		Body      string
		ContactJ  string
		CName     string
	}
	var tl []tlRow
	if err := db.Raw(`
		SELECT m.created_at, m.direction, m.body, c.contact_j_id AS contact_j, COALESCE(NULLIF(TRIM(c.contact_name), ''), '') AS c_name
		FROM messages m
		INNER JOIN conversations c ON c.id = m.conversation_id
		ORDER BY m.created_at DESC
		LIMIT ?`, nt).Scan(&tl).Error; err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(tl) == 0 {
		fmt.Println("(nenhuma mensagem na inbox — ainda não há conversas gravadas)")
	} else {
		for _, r := range tl {
			label := "[recebida]"
			if r.Direction == "outbound" {
				label = "[enviada]  "
			}
			who := r.ContactJ
			if r.CName != "" {
				who = fmt.Sprintf("%s (%s)", r.CName, r.ContactJ)
			}
			fmt.Printf("%s  %s  %s  %q\n",
				r.CreatedAt.UTC().Format("2006-01-02 15:04:05"), label, who, truncateRunes(r.Body, 200))
		}
	}

	fmt.Printf("\n=== Histórico por conversa (últimas %d conversas ativas, até %d mensagens cada, ordem cronológica) ===\n", nth, nm)
	var convs []model.Conversation
	if err := db.Order("last_message_at DESC NULLS LAST").Order("updated_at DESC").Limit(nth).Find(&convs).Error; err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(convs) == 0 {
		fmt.Println("(sem conversas na BD)")
	} else {
		for _, c := range convs {
			title := c.ContactJID
			if nm := strings.TrimSpace(c.ContactName); nm != "" {
				title = fmt.Sprintf("%s — %s", nm, c.ContactJID)
			}
			fmt.Printf("\n— Conversa: %s —\n", title)
			var msgs []model.Message
			if err := db.Where("conversation_id = ?", c.ID).Order("created_at DESC").Limit(nm).Find(&msgs).Error; err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			if len(msgs) == 0 {
				fmt.Println("  (sem mensagens ligadas)")
				continue
			}
			for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
				msgs[i], msgs[j] = msgs[j], msgs[i]
			}
			for _, m := range msgs {
				label := "←"
				if m.Direction == "outbound" {
					label = "→"
				}
				fmt.Printf("  %s %s  %q\n", m.CreatedAt.UTC().Format("15:04:05"), label, truncateRunes(m.Body, 300))
			}
		}
	}

	fmt.Println("\n=== Últimas 15 mensagens só recebidas (atalho) ===")
	type row struct {
		ID        string
		CreatedAt time.Time
		Body      string
		ContactJ  string
	}
	var inbox []row
	if err := db.Raw(`
		SELECT m.id::text, m.created_at, m.body, c.contact_j_id AS contact_j
		FROM messages m
		INNER JOIN conversations c ON c.id = m.conversation_id
		WHERE m.direction = 'inbound'
		ORDER BY m.created_at DESC
		LIMIT 15`).Scan(&inbox).Error; err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(inbox) == 0 {
		fmt.Println("(nenhuma inbound na tabela messages)")
	} else {
		for _, r := range inbox {
			fmt.Printf("%s  contact=%s body=%q\n", r.CreatedAt.UTC().Format(time.RFC3339), r.ContactJ, r.Body)
		}
	}

	if t := strings.TrimSpace(o.textSnippet); t != "" {
		tl := strings.ToLower(t)
		fmt.Printf("\n=== Procurando %q em mensagens (recebidas e enviadas) ===\n", t)
		type hit struct {
			ID        uuid.UUID
			CreatedAt time.Time
			Body      string
			Direction string
			ContactJ  string
		}
		var hits []hit
		_ = db.Raw(`
			SELECT m.id, m.created_at, m.body, m.direction, c.contact_j_id AS contact_j
			FROM messages m
			INNER JOIN conversations c ON c.id = m.conversation_id
			WHERE LOWER(m.body) LIKE ?
			ORDER BY m.created_at DESC
			LIMIT 10`, "%"+tl+"%").Scan(&hits).Error
		if len(hits) == 0 {
			fmt.Println("(nenhum hit na inbox — tenta o re-parse nos webhooks acima ou envia mensagem de novo com a API atualizada)")
		} else {
			for _, h := range hits {
				fmt.Printf("  [%s] %s  %s  %q\n", h.Direction, h.CreatedAt.UTC().Format(time.RFC3339), h.ContactJ, h.Body)
			}
		}
	}

	if len(wms) > 0 {
		slug := strings.TrimSpace(wms[0].InstanceID)
		if slug != "" {
			fmt.Println("\n=== Webhooks na auditoria mas sem inbound na tabela messages? ===")
			fmt.Printf("Reprocessa e grava: go run ./cmd/waitinbound -reconcile -instance %q\n", slug)
		}
	}
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
