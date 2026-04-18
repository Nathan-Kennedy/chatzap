package handler

import (
	"context"
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

type sendMessageBody struct {
	Body string `json:"body"`
}

// messageRowHasRetrievableAttachment alinha com HandleGetMessageAttachment: só true se existir
// ficheiro local, URL remota, ou dados Evolution (JSON da mensagem ou key.id) para tipos não-texto.
func messageRowHasRetrievableAttachment(m model.Message) bool {
	if strings.TrimSpace(m.StoredMediaPath) != "" || strings.TrimSpace(m.MediaRemoteURL) != "" {
		return true
	}
	mt := strings.TrimSpace(m.MessageType)
	if mt == "" {
		mt = "text"
	}
	if mt == "text" {
		return false
	}
	return strings.TrimSpace(m.WaMediaMessageJSON) != "" || strings.TrimSpace(m.ExternalID) != ""
}

type createConversationBody struct {
	WhatsAppInstanceID string `json:"whatsapp_instance_id"`
	Phone              string `json:"phone"`
	ContactJID         string `json:"contact_jid"`
	ContactName        string `json:"contact_name"`
}

// HandleCreateConversation POST /api/v1/conversations — abre thread sem mensagem (operador inicia contacto).
func HandleCreateConversation(db *gorm.DB, rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}

		var body createConversationBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		iid, err := uuid.Parse(strings.TrimSpace(body.WhatsAppInstanceID))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "whatsapp_instance_id inválido", nil)
		}

		var jid string
		if strings.TrimSpace(body.ContactJID) != "" {
			jid = service.NormalizeContactJID(body.ContactJID)
		} else {
			jid = service.NormalizeContactJID(body.Phone)
		}
		if jid == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "phone ou contact_jid obrigatório", nil)
		}

		var inst model.WhatsAppInstance
		if err := db.Where("id = ? AND workspace_id = ?", iid, wid).First(&inst).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return JSONError(c, fiber.StatusNotFound, "not_found", "instância não encontrada neste workspace", nil)
			}
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		var existing model.Conversation
		err = db.Where("workspace_id = ? AND whats_app_instance_id = ? AND contact_j_id = ?", wid, iid, jid).First(&existing).Error
		if err == nil {
			return JSONError(c, fiber.StatusConflict, "conversation_exists", "conversa já existe", fiber.Map{
				"conversation": mapConversation(&existing),
			})
		}
		if err != gorm.ErrRecordNotFound {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		now := time.Now()
		display := strings.TrimSpace(body.ContactName)
		if display == "" {
			display = service.DisplayNameFromJID(jid)
		}

		conv := model.Conversation{
			WorkspaceID:        wid,
			WhatsAppInstanceID: iid,
			ContactJID:         jid,
			ContactName:        display,
			LastMessageAt:      now,
			LastMessagePreview: "Nova conversa",
			Channel:            "whatsapp",
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := db.Create(&conv).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		service.PublishInboxEvent(rdb, wid, map[string]interface{}{
			"type":            "conversation.updated",
			"conversation_id": conv.ID.String(),
		})

		return JSONSuccess(c, mapConversation(&conv))
	}
}

// HandleDeleteConversation DELETE /api/v1/conversations/:id — apaga mensagens e a conversa (workspace).
func HandleDeleteConversation(db *gorm.DB, rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		cid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}

		var conv model.Conversation
		if err := db.Where("id = ? AND workspace_id = ?", cid, wid).First(&conv).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return JSONError(c, fiber.StatusNotFound, "not_found", "conversa não encontrada", nil)
			}
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("conversation_id = ?", cid).Delete(&model.Message{}).Error; err != nil {
				return err
			}
			return tx.Where("id = ? AND workspace_id = ?", cid, wid).Delete(&model.Conversation{}).Error
		})
		if err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		service.PublishInboxEvent(rdb, wid, map[string]interface{}{
			"type":            "conversation.deleted",
			"conversation_id": cid.String(),
		})

		return JSONSuccess(c, fiber.Map{"ok": true})
	}
}

// HandleListConversations GET /api/v1/conversations
func HandleListConversations(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		q := strings.TrimSpace(c.Query("search"))

		tx := db.Model(&model.Conversation{}).Where("workspace_id = ?", wid).
			Order("COALESCE(last_message_at, created_at) DESC").Limit(200)
		if q != "" {
			like := "%" + strings.ToLower(q) + "%"
			tx = tx.Where("LOWER(contact_name) LIKE ? OR LOWER(contact_j_id) LIKE ?", like, like)
		}

		var rows []model.Conversation
		if err := tx.Find(&rows).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		out := make([]fiber.Map, 0, len(rows))
		for _, r := range rows {
			out = append(out, mapConversation(&r))
		}
		return JSONSuccess(c, out)
	}
}

func mapConversation(r *model.Conversation) fiber.Map {
	phone := r.ContactJID
	if i := strings.Index(phone, "@"); i > 0 {
		phone = phone[:i]
	}
	return fiber.Map{
		"id":                      r.ID.String(),
		"whatsapp_instance_id":    r.WhatsAppInstanceID.String(),
		"contact_id":              r.ContactJID,
		"contact_name":            r.ContactName,
		"contact_phone":           phone,
		"channel":                 r.Channel,
		"last_message_preview":    r.LastMessagePreview,
		"unread_count":            0,
		"updated_at":              r.UpdatedAt.UTC().Format(time.RFC3339Nano),
		"assigned_agent_initials": r.AssignedAgentInitials,
		"status":                  "open",
	}
}

// HandleListMessages GET /api/v1/conversations/:id/messages
func HandleListMessages(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		cid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}

		var conv model.Conversation
		if err := db.Where("id = ? AND workspace_id = ?", cid, wid).First(&conv).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "conversa não encontrada", nil)
		}

		var msgs []model.Message
		if err := db.Where("conversation_id = ?", cid).Order("created_at ASC").Limit(500).Find(&msgs).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		out := make([]fiber.Map, 0, len(msgs))
		for _, m := range msgs {
			dir := m.Direction
			if dir == "inbound" || dir == "outbound" {
				// ok
			} else {
				dir = "inbound"
			}
			mt := strings.TrimSpace(m.MessageType)
			if mt == "" {
				mt = "text"
			}
			hasAttach := messageRowHasRetrievableAttachment(m)
			out = append(out, fiber.Map{
				"id":              m.ID.String(),
				"conversation_id": cid.String(),
				"direction":       dir,
				"body":            m.Body,
				"message_type":    mt,
				"file_name":       m.FileName,
				"mime_type":       m.MimeType,
				"has_attachment":  hasAttach,
				"created_at":      m.CreatedAt.UTC().Format(time.RFC3339Nano),
			})
		}
		return JSONSuccess(c, out)
	}
}

// HandleSendMessage POST /api/v1/conversations/:id/messages
func HandleSendMessage(log *zap.Logger, db *gorm.DB, rdb *redis.Client, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg.WhatsAppProvider != "evolution" || ev == nil {
			return JSONError(c, fiber.StatusServiceUnavailable, "evolution_not_configured", "Evolution necessário para enviar", nil)
		}
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		cid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}

		var body sendMessageBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		text := strings.TrimSpace(body.Body)
		if text == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "body obrigatório", nil)
		}

		var conv model.Conversation
		if err := db.Where("id = ? AND workspace_id = ?", cid, wid).First(&conv).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "conversa não encontrada", nil)
		}

		var inst model.WhatsAppInstance
		if err := db.Where("id = ?", conv.WhatsAppInstanceID).First(&inst).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", "instância não encontrada", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
		defer cancel()
		sendRaw, err := ev.SendText(ctx, inst.EvolutionInstanceToken, conv.ContactJID, text)
		if err != nil {
			log.Error("send text", zap.Error(err))
			return JSONError(c, fiber.StatusBadGateway, "send_failed", err.Error(), nil)
		}

		sendRJ, sendKeyID := service.ParseEvolutionSendTextResponse(sendRaw)
		if strings.TrimSpace(sendRJ) == "" {
			sendRJ = conv.ContactJID
		}

		if err := service.RecordOutboundMessage(db, cid, text, sendKeyID); err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		if err := service.PersistPortalOutboundWebhook(db, inst.EvolutionInstanceName, sendRJ, "", text, sendKeyID); err != nil {
			log.Warn("portal outbound webhook audit", zap.Error(err))
		}

		service.PublishInboxEvent(rdb, wid, map[string]interface{}{
			"type":            "message.created",
			"conversation_id": cid.String(),
		})

		return JSONSuccess(c, fiber.Map{"ok": true})
	}
}
