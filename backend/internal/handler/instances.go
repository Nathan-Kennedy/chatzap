package handler

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
	"wa-saas/backend/internal/service"
)

var evolutionNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,62}$`)

type createInstanceBody struct {
	EvolutionInstanceName string `json:"evolution_instance_name"`
	DisplayName           string `json:"display_name"`
}

type importInstanceBody struct {
	EvolutionInstanceName  string `json:"evolution_instance_name"`
	EvolutionInstanceToken string `json:"evolution_instance_token"`
	DisplayName            string `json:"display_name"`
}

func syncEvolutionWebhook(ctx context.Context, log *zap.Logger, cfg *config.Config, ev *service.EvolutionClient, row *model.WhatsAppInstance) {
	u := cfg.WebhookURLForWhatsAppInstance(row.EvolutionInstanceName)
	if u == "" || ev == nil {
		return
	}
	if err := ev.SetInstanceWebhook(ctx, row.EvolutionInstanceName, row.EvolutionInstanceToken, u); err != nil {
		log.Warn("evolution webhook sync falhou",
			zap.String("instance", row.EvolutionInstanceName),
			zap.String("webhook_url", u),
			zap.Error(err),
		)
	}
}

func mapInstanceRow(m *model.WhatsAppInstance, messagesToday int64) fiber.Map {
	num := m.PhoneE164
	if num == "" {
		num = "—"
	}
	return fiber.Map{
		"id":                      m.ID.String(),
		"name":                    m.DisplayName,
		"evolution_instance_name": m.EvolutionInstanceName,
		"number":                  num,
		"status":                  m.Status,
		"messages_today":          messagesToday,
	}
}

// HandleListInstances GET /api/v1/instances
func HandleListInstances(_ *zap.Logger, db *gorm.DB, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.WhatsAppProvider != "evolution" || ev == nil {
			return JSONError(c, fiber.StatusServiceUnavailable, "evolution_not_configured", "defina WHATSAPP_PROVIDER=evolution e Evolution no Docker", nil)
		}
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "workspace inválido", nil)
		}

		var rows []model.WhatsAppInstance
		if err := db.Where("workspace_id = ?", wid).Order("created_at DESC").Find(&rows).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 20*time.Second)
		defer cancel()
		remote, _ := ev.FetchInstances(ctx)
		remoteByName := make(map[string]service.EvolutionInstanceInfo)
		for _, r := range remote {
			if r.Name != "" {
				remoteByName[strings.ToLower(r.Name)] = r
			}
		}

		out := make([]fiber.Map, 0, len(rows))
		for i := range rows {
			row := &rows[i]
			var n int64
			_ = db.Table("messages").
				Joins("INNER JOIN conversations ON conversations.id = messages.conversation_id").
				Where("conversations.whats_app_instance_id = ? AND messages.created_at >= ?", row.ID, startOfTodayUTC()).
				Count(&n).Error

			if ri, ok := remoteByName[strings.ToLower(row.EvolutionInstanceName)]; ok {
				row.Status = mapEvolutionStatus(ri.Status, boolToConnState(ri.Connected))
				row.PhoneE164 = firstNonEmpty(ri.OwnerJID, row.PhoneE164)
				_ = db.Model(row).Updates(map[string]interface{}{
					"status":     row.Status,
					"phone_e164": nullIfEmpty(row.PhoneE164),
				}).Error
			}

			out = append(out, mapInstanceRow(row, n))
		}

		return JSONSuccess(c, out)
	}
}

func startOfTodayUTC() time.Time {
	t := time.Now().UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func mapEvolutionStatus(s, conn string) string {
	x := strings.ToLower(strings.TrimSpace(s + " " + conn))
	if strings.Contains(x, "open") || strings.Contains(x, "connected") {
		return "connected"
	}
	if strings.Contains(x, "close") || strings.Contains(x, "disconnect") {
		return "disconnected"
	}
	if strings.Contains(x, "connect") || strings.Contains(x, "qr") {
		return "qr_pending"
	}
	return "qr_pending"
}

func boolToConnState(v bool) string {
	if v {
		return "connected"
	}
	return "disconnected"
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func nullIfEmpty(s string) string {
	return s
}

// HandleCreateInstance POST /api/v1/instances
func HandleCreateInstance(log *zap.Logger, db *gorm.DB, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.WhatsAppProvider != "evolution" || ev == nil {
			return JSONError(c, fiber.StatusServiceUnavailable, "evolution_not_configured", "Evolution não configurado", nil)
		}
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "workspace inválido", nil)
		}

		var body createInstanceBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		name := strings.TrimSpace(strings.ToLower(body.EvolutionInstanceName))
		if !evolutionNameRe.MatchString(name) {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "evolution_instance_name: use 2–63 caracteres [a-z0-9_-] começando por letra ou número", nil)
		}
		display := strings.TrimSpace(body.DisplayName)
		if display == "" {
			display = name
		}

		var existing int64
		if err := db.Model(&model.WhatsAppInstance{}).Where("evolution_instance_name = ?", name).Count(&existing).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		if existing > 0 {
			return JSONError(c, fiber.StatusConflict, "instance_exists", "nome de instância já existe", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()
		instanceToken := uuid.NewString()
		if err := ev.CreateInstance(ctx, name, instanceToken); err != nil {
			log.Error("evolution create", zap.Error(err))
			return JSONError(c, fiber.StatusBadGateway, "evolution_error", err.Error(), nil)
		}

		row := model.WhatsAppInstance{
			WorkspaceID:            wid,
			EvolutionInstanceName:  name,
			EvolutionInstanceToken: instanceToken,
			DisplayName:            display,
			Status:                 "qr_pending",
		}
		if err := db.Create(&row).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		syncEvolutionWebhook(ctx, log, cfg, ev, &row)

		var n int64
		return JSONSuccess(c, mapInstanceRow(&row, n))
	}
}

// HandleImportInstance POST /api/v1/instances/import — instância já criada no Manager/Evolution.
func HandleImportInstance(log *zap.Logger, db *gorm.DB, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.WhatsAppProvider != "evolution" || ev == nil {
			return JSONError(c, fiber.StatusServiceUnavailable, "evolution_not_configured", "Evolution não configurado", nil)
		}
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "workspace inválido", nil)
		}

		var body importInstanceBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		name := strings.TrimSpace(strings.ToLower(body.EvolutionInstanceName))
		if !evolutionNameRe.MatchString(name) {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "evolution_instance_name: use 2–63 caracteres [a-z0-9_-] começando por letra ou número", nil)
		}
		token := strings.TrimSpace(body.EvolutionInstanceToken)
		if token == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "evolution_instance_token obrigatório", nil)
		}
		display := strings.TrimSpace(body.DisplayName)
		if display == "" {
			display = name
		}

		var existing int64
		if err := db.Model(&model.WhatsAppInstance{}).Where("evolution_instance_name = ?", name).Count(&existing).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		if existing > 0 {
			return JSONError(c, fiber.StatusConflict, "instance_exists", "nome de instância já existe", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()

		remote, err := ev.FetchInstances(ctx)
		if err != nil {
			log.Error("evolution list", zap.Error(err))
			return JSONError(c, fiber.StatusBadGateway, "evolution_error", err.Error(), nil)
		}
		found := false
		for _, r := range remote {
			if strings.EqualFold(strings.TrimSpace(r.Name), name) {
				found = true
				break
			}
		}
		if !found {
			return JSONError(c, fiber.StatusNotFound, "evolution_instance_not_found", "instância não encontrada na Evolution", nil)
		}

		if _, err := ev.ConnectionState(ctx, token); err != nil {
			log.Error("evolution token/status", zap.Error(err))
			return JSONError(c, fiber.StatusBadRequest, "invalid_token", "token inválido ou instância inacessível", nil)
		}

		row := model.WhatsAppInstance{
			WorkspaceID:            wid,
			EvolutionInstanceName:  name,
			EvolutionInstanceToken: token,
			DisplayName:            display,
			Status:                 "connected",
		}
		if err := db.Create(&row).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		syncEvolutionWebhook(ctx, log, cfg, ev, &row)

		var n int64
		return JSONSuccess(c, mapInstanceRow(&row, n))
	}
}

// HandleSyncInstanceWebhook POST /api/v1/instances/:id/sync-webhook — reconfigura webhook na Evolution.
func HandleSyncInstanceWebhook(log *zap.Logger, db *gorm.DB, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.WhatsAppProvider != "evolution" || ev == nil {
			return JSONError(c, fiber.StatusServiceUnavailable, "evolution_not_configured", "Evolution não configurado", nil)
		}
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "workspace inválido", nil)
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}

		var row model.WhatsAppInstance
		if err := db.Where("id = ? AND workspace_id = ?", id, wid).First(&row).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "instância não encontrada", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()

		u := cfg.WebhookURLForWhatsAppInstance(row.EvolutionInstanceName)
		if u == "" {
			return JSONError(c, fiber.StatusInternalServerError, "config_error", "PUBLIC_WEBHOOK_BASE_URL inválido", nil)
		}
		if err := ev.SetInstanceWebhook(ctx, row.EvolutionInstanceName, row.EvolutionInstanceToken, u); err != nil {
			log.Error("evolution set webhook", zap.Error(err))
			return JSONError(c, fiber.StatusBadGateway, "evolution_error", err.Error(), nil)
		}
		return JSONSuccess(c, fiber.Map{"ok": true, "webhook_url": u})
	}
}

type syncChatsBody struct {
	Phone         string `json:"phone"`
	ContactJID    string `json:"contact_jid"`
	ContactJIDAlt string `json:"contact_jid_alt"` // ex. @lid quando a conversa está no número
}

// HandleSyncChatHistory POST /api/v1/instances/:id/sync-chats — importa histórico via Evolution /chat/findMessages (se existir).
func HandleSyncChatHistory(log *zap.Logger, db *gorm.DB, rdb *redis.Client, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.WhatsAppProvider != "evolution" || ev == nil {
			return JSONError(c, fiber.StatusServiceUnavailable, "evolution_not_configured", "Evolution não configurado", nil)
		}
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "workspace inválido", nil)
		}
		iid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}

		var body syncChatsBody
		_ = c.BodyParser(&body)
		var jid string
		if strings.TrimSpace(body.ContactJID) != "" {
			jid = service.NormalizeContactJID(body.ContactJID)
		} else {
			jid = service.NormalizeContactJID(body.Phone)
		}
		if jid == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "phone ou contact_jid obrigatório", nil)
		}
		altNorm := ""
		if strings.TrimSpace(body.ContactJIDAlt) != "" {
			altNorm = service.NormalizeContactJID(body.ContactJIDAlt)
		}
		lookupKeys := service.CollectJIDLookupKeys(jid, altNorm)

		var inst model.WhatsAppInstance
		if err := db.Where("id = ? AND workspace_id = ?", iid, wid).First(&inst).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "instância não encontrada", nil)
		}

		var conv model.Conversation
		found := false
		for _, k := range lookupKeys {
			convErr := db.Where("workspace_id = ? AND whats_app_instance_id = ? AND contact_j_id = ?", wid, iid, k).First(&conv).Error
			if convErr == nil {
				found = true
				break
			}
			if convErr != gorm.ErrRecordNotFound {
				return JSONError(c, fiber.StatusInternalServerError, "db_error", convErr.Error(), nil)
			}
		}
		canonicalJID := lookupKeys[0]
		if !found {
			now := time.Now()
			conv = model.Conversation{
				WorkspaceID:        wid,
				WhatsAppInstanceID: iid,
				ContactJID:         canonicalJID,
				ContactName:        service.DisplayNameFromJID(canonicalJID),
				LastMessageAt:      now,
				LastMessagePreview: "Importação",
				Channel:            "whatsapp",
				CreatedAt:          now,
				UpdatedAt:          now,
			}
			if err := db.Create(&conv).Error; err != nil {
				return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
			}
		}

		ctx, cancel := context.WithTimeout(c.Context(), 90*time.Second)
		defer cancel()

		seenItem := make(map[string]struct{})
		var items []service.HistoryImportItem
		var lastErr string
		var anyOK bool
		for _, k := range lookupKeys {
			status, raw, err := ev.ChatFindMessages(ctx, inst.EvolutionInstanceName, inst.EvolutionInstanceToken, k)
			if err != nil {
				return JSONError(c, fiber.StatusBadGateway, "evolution_error", err.Error(), nil)
			}
			if status == http.StatusNotFound {
				return JSONError(c, fiber.StatusNotImplemented, "sync_not_supported",
					"Esta build do Evolution Go não expõe POST /chat/findMessages (404). O histórico em tempo real usa webhooks.", nil)
			}
			if status < 200 || status >= 300 {
				lastErr = fmt.Sprintf("findMessages(%s) status %d: %s", k, status, string(raw))
				continue
			}
			anyOK = true
			batch, err := service.ParseFindMessagesResponse(raw)
			if err != nil {
				log.Warn("parse findMessages", zap.String("jid", k), zap.Error(err))
				lastErr = err.Error()
				continue
			}
			for _, it := range batch {
				dedupeKey := it.ExternalID
				if dedupeKey == "" {
					dedupeKey = fmt.Sprintf("%s|%d|%s", it.Body, it.CreatedAt.Unix(), it.Direction)
				}
				if _, ok := seenItem[dedupeKey]; ok {
					continue
				}
				seenItem[dedupeKey] = struct{}{}
				items = append(items, it)
			}
		}
		if !anyOK && lastErr != "" {
			return JSONError(c, fiber.StatusBadGateway, "evolution_error", lastErr, nil)
		}

		sort.Slice(items, func(i, j int) bool {
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		})

		inserted := 0
		for _, it := range items {
			ok, err := service.InsertHistoryMessageIfNew(db, conv.ID, it)
			if err != nil {
				return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
			}
			if ok {
				inserted++
			}
		}

		if inserted > 0 {
			if err := service.RefreshConversationPreview(db, conv.ID); err != nil {
				log.Warn("refresh preview", zap.Error(err))
			}
			service.PublishInboxEvent(rdb, wid, map[string]interface{}{
				"type":            "conversation.updated",
				"conversation_id": conv.ID.String(),
			})
		}

		return JSONSuccess(c, fiber.Map{
			"inserted":        inserted,
			"parsed":          len(items),
			"conversation_id": conv.ID.String(),
		})
	}
}

type reconcileInboxBody struct {
	BodyContains string `json:"body_contains"`
	Limit        int    `json:"limit"`
}

// HandleReconcileInboxWebhooks POST /api/v1/instances/:id/reconcile-inbox — reprocessa webhook_messages para a caixa (ex. mensagem recebida mas não associada).
func HandleReconcileInboxWebhooks(log *zap.Logger, db *gorm.DB, rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "workspace inválido", nil)
		}
		iid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}

		var body reconcileInboxBody
		_ = c.BodyParser(&body)

		var inst model.WhatsAppInstance
		if err := db.Where("id = ? AND workspace_id = ?", iid, wid).First(&inst).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "instância não encontrada", nil)
		}

		n, scanned, err := service.ReconcileWebhooksToInbox(db, log, rdb, wid, inst, body.BodyContains, body.Limit)
		if err != nil {
			log.Error("reconcile inbox", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		return JSONSuccess(c, fiber.Map{
			"new_messages": n,
			"scanned":      scanned,
		})
	}
}

func truncateContactNameField(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	r := []rune(s)
	if len(r) <= 500 {
		return s
	}
	return string(r[:500])
}

// HandleSyncWhatsAppContacts POST /api/v1/instances/:id/sync-contacts — cria conversas a partir de GET /user/contacts na Evolution (contactos/chats no telefone).
// Não importa o texto das mensagens antigas; isso depende de webhooks ou de sync-chats por contacto (findMessages).
func HandleSyncWhatsAppContacts(log *zap.Logger, db *gorm.DB, rdb *redis.Client, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.WhatsAppProvider != "evolution" || ev == nil {
			return JSONError(c, fiber.StatusServiceUnavailable, "evolution_not_configured", "Evolution não configurado", nil)
		}
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "workspace inválido", nil)
		}
		iid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}

		var inst model.WhatsAppInstance
		if err := db.Where("id = ? AND workspace_id = ?", iid, wid).First(&inst).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "instância não encontrada", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 120*time.Second)
		defer cancel()

		remote, err := ev.FetchWhatsAppContacts(ctx, inst.EvolutionInstanceToken)
		if err != nil {
			log.Error("evolution fetch contacts", zap.Error(err))
			return JSONError(c, fiber.StatusBadGateway, "evolution_error", err.Error(), nil)
		}

		created := 0
		skipped := 0
		already := 0
		now := time.Now()

		err = db.Transaction(func(tx *gorm.DB) error {
			for _, row := range remote {
				rawJ := strings.TrimSpace(row.Jid)
				if rawJ == "" || strings.Contains(strings.ToLower(rawJ), "broadcast") {
					skipped++
					continue
				}
				jid := service.NormalizeContactJID(rawJ)
				if jid == "" {
					skipped++
					continue
				}

				var ex model.Conversation
				q := tx.Where("workspace_id = ? AND whats_app_instance_id = ? AND contact_j_id = ?", wid, iid, jid)
				if err := q.First(&ex).Error; err == nil {
					already++
					continue
				}
				if err != gorm.ErrRecordNotFound {
					return err
				}

				disp := truncateContactNameField(service.ContactDisplayNameFromEvolution(row))
				if disp == "" {
					disp = service.DisplayNameFromJID(jid)
				}

				conv := model.Conversation{
					WorkspaceID:        wid,
					WhatsAppInstanceID: iid,
					ContactJID:         jid,
					ContactName:        disp,
					LastMessageAt:      now,
					LastMessagePreview: "Importado do WhatsApp",
					Channel:            "whatsapp",
					CreatedAt:          now,
					UpdatedAt:          now,
				}
				if err := tx.Create(&conv).Error; err != nil {
					return err
				}
				created++
			}
			return nil
		})
		if err != nil {
			log.Error("sync contacts tx", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		if created > 0 {
			service.PublishInboxEvent(rdb, wid, map[string]interface{}{
				"type": "conversations.imported",
			})
		}

		return JSONSuccess(c, fiber.Map{
			"total_fetched":     len(remote),
			"created":         created,
			"already_existing": already,
			"skipped":         skipped,
		})
	}
}

// HandleInstanceQRCode GET /api/v1/instances/:id/qrcode
func HandleInstanceQRCode(log *zap.Logger, db *gorm.DB, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.WhatsAppProvider != "evolution" || ev == nil {
			return JSONError(c, fiber.StatusServiceUnavailable, "evolution_not_configured", "Evolution não configurado", nil)
		}
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}

		var row model.WhatsAppInstance
		if err := db.Where("id = ? AND workspace_id = ?", id, wid).First(&row).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "instância não encontrada", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()
		conn, err := ev.ConnectInstance(ctx, row.EvolutionInstanceToken)
		if err != nil {
			log.Error("evolution connect", zap.Error(err))
			return JSONError(c, fiber.StatusBadGateway, "evolution_error", err.Error(), nil)
		}
		code := service.NormalizeQRDataURLForBrowser(conn.Code)
		pair := strings.TrimSpace(conn.PairingCode)

		if code == "" && pair == "" {
			st, stErr := ev.ConnectionState(ctx, row.EvolutionInstanceToken)
			if stErr == nil {
				low := strings.ToLower(st)
				if strings.Contains(low, "connect") && !strings.Contains(low, "disconnect") && !strings.Contains(low, "qr") {
					return JSONSuccess(c, fiber.Map{
						"code":              "",
						"pairing_code":      "",
						"already_connected": true,
						"instance_name":     row.EvolutionInstanceName,
						"evolution_status":  st,
					})
				}
			}
			return JSONError(c, fiber.StatusBadGateway, "evolution_qr_empty",
				"A Evolution não devolveu imagem de QR nem código de pareamento. Se já estás ligado, ignora; senão, tenta na Evolution Manager ou recria a sessão.", nil)
		}

		return JSONSuccess(c, fiber.Map{
			"code":              code,
			"pairing_code":      pair,
			"instance_name":     row.EvolutionInstanceName,
			"already_connected": false,
		})
	}
}

// HandleGetInstance GET /api/v1/instances/:id
func HandleGetInstance(db *gorm.DB, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.WhatsAppProvider != "evolution" || ev == nil {
			return JSONError(c, fiber.StatusServiceUnavailable, "evolution_not_configured", "", nil)
		}
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}
		var row model.WhatsAppInstance
		if err := db.Where("id = ? AND workspace_id = ?", id, wid).First(&row).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "", nil)
		}
		var n int64
		_ = db.Table("messages").
			Joins("INNER JOIN conversations ON conversations.id = messages.conversation_id").
			Where("conversations.whats_app_instance_id = ? AND messages.created_at >= ?", row.ID, startOfTodayUTC()).
			Count(&n).Error
		return JSONSuccess(c, mapInstanceRow(&row, n))
	}
}

// HandleDeleteInstance DELETE /api/v1/instances/:id — remove na Evolution (best-effort) e apaga conversas/mensagens na BD.
func HandleDeleteInstance(log *zap.Logger, db *gorm.DB, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "workspace inválido", nil)
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}

		var row model.WhatsAppInstance
		if err := db.Where("id = ? AND workspace_id = ?", id, wid).First(&row).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return JSONError(c, fiber.StatusNotFound, "not_found", "instância não encontrada", nil)
			}
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()
		if cfg.WhatsAppProvider == "evolution" && ev != nil && strings.TrimSpace(cfg.EvolutionBaseURL) != "" {
			if err := ev.DeleteRemoteInstance(ctx, row.EvolutionInstanceName); err != nil {
				log.Warn("evolution delete instance (remove na mesma na API SaaS)",
					zap.String("name", row.EvolutionInstanceName),
					zap.Error(err),
				)
			}
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			sub := tx.Model(&model.Conversation{}).Select("id").Where("whats_app_instance_id = ?", row.ID)
			if err := tx.Where("conversation_id IN (?)", sub).Delete(&model.Message{}).Error; err != nil {
				return err
			}
			if err := tx.Where("whats_app_instance_id = ?", row.ID).Delete(&model.Conversation{}).Error; err != nil {
				return err
			}
			_ = tx.Where("instance_id = ?", row.EvolutionInstanceName).Delete(&model.WebhookMessage{}).Error
			return tx.Delete(&row).Error
		})
		if err != nil {
			log.Error("delete instance", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		return JSONSuccess(c, fiber.Map{"ok": true})
	}
}
