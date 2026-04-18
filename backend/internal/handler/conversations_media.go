package handler

import (
	"context"
	"errors"
	"mime"
	"path/filepath"
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

var allowedMediaTypes = map[string][]string{
	"image":    {"image/jpeg", "image/png", "image/gif", "image/webp"},
	"video":    {"video/mp4", "video/quicktime", "video/webm"},
	"audio":    {"audio/webm", "audio/ogg", "audio/mpeg", "audio/mp4", "audio/wav", "audio/x-wav", "application/ogg"},
	"document": nil, // qualquer MIME
}

func mimeAllowed(mediaKind, mimeType string) bool {
	mimeType = strings.TrimSpace(strings.ToLower(mimeType))
	list := allowedMediaTypes[strings.ToLower(mediaKind)]
	if list == nil {
		return true
	}
	for _, a := range list {
		if mimeType == a {
			return true
		}
	}
	return false
}

// HandleSendConversationMedia POST /api/v1/conversations/:id/messages/media (multipart: file, type, caption opcional).
func HandleSendConversationMedia(log *zap.Logger, db *gorm.DB, rdb *redis.Client, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
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

		mediaKind := strings.ToLower(strings.TrimSpace(c.FormValue("type")))
		switch mediaKind {
		case "image", "video", "audio", "document":
		default:
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "type deve ser image, video, audio ou document", nil)
		}
		caption := strings.TrimSpace(c.FormValue("caption"))

		fh, err := c.FormFile("file")
		if err != nil || fh == nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "campo file obrigatório", nil)
		}
		if fh.Size > cfg.MediaMaxUploadBytes {
			return JSONError(c, fiber.StatusRequestEntityTooLarge, "file_too_large", "ficheiro acima do limite configurado", nil)
		}

		f, err := fh.Open()
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "não foi possível ler o ficheiro", nil)
		}
		defer f.Close()

		mimeType := fh.Header.Get("Content-Type")
		if mimeType == "" || mimeType == "application/octet-stream" {
			if ext := filepath.Ext(fh.Filename); ext != "" {
				if mt := mime.TypeByExtension(ext); mt != "" {
					mimeType = mt
				}
			}
		}
		if !mimeAllowed(mediaKind, mimeType) {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "tipo MIME não permitido para "+mediaKind, nil)
		}

		var conv model.Conversation
		if err := db.Where("id = ? AND workspace_id = ?", cid, wid).First(&conv).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "conversa não encontrada", nil)
		}

		var inst model.WhatsAppInstance
		if err := db.Where("id = ?", conv.WhatsAppInstanceID).First(&inst).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", "instância não encontrada", nil)
		}

		ttl := time.Duration(cfg.MediaTokenTTLMinutes) * time.Minute
		if ttl <= 0 {
			ttl = 30 * time.Minute
		}

		plainToken, tokRow, err := service.NewMediaTempToken(db, cfg.MediaUploadDir, ttl, fh.Filename, mimeType, f, cfg.MediaMaxUploadBytes)
		if err != nil {
			if errors.Is(err, service.ErrMediaTooLarge) {
				return JSONError(c, fiber.StatusRequestEntityTooLarge, "file_too_large", err.Error(), nil)
			}
			log.Error("media temp token", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		publicURL := cfg.MediaTempFetchURL(plainToken)
		if publicURL == "" {
			service.DeleteMediaTempToken(db, tokRow)
			return JSONError(c, fiber.StatusInternalServerError, "config_error", "PUBLIC_MEDIA_BASE_URL / PUBLIC_WEBHOOK_BASE_URL em falta", nil)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 120*time.Second)
		defer cancel()

		docName := fh.Filename
		if mediaKind != "document" {
			docName = ""
		}
		sendRaw, err := ev.SendMedia(ctx, inst.EvolutionInstanceToken, conv.ContactJID, mediaKind, publicURL, caption, docName)
		if err != nil {
			service.DeleteMediaTempToken(db, tokRow)
			log.Error("send media", zap.String("kind", mediaKind), zap.Error(err))
			return JSONError(c, fiber.StatusBadGateway, "send_failed", err.Error(), nil)
		}

		sendRJ, sendKeyID := service.ParseEvolutionSendTextResponse(sendRaw)
		if strings.TrimSpace(sendRJ) == "" {
			sendRJ = conv.ContactJID
		}

		body := caption
		if body == "" {
			switch mediaKind {
			case "image":
				body = "[imagem]"
			case "video":
				body = "[vídeo]"
			case "audio":
				body = "[áudio]"
			case "document":
				body = "[documento]"
				if fh.Filename != "" {
					body = fh.Filename
				}
			}
		}

		msgID, err := service.RecordOutbound(db, cid, service.OutboundRecord{
			Body:         body,
			ExternalID:   sendKeyID,
			MessageType:  mediaKind,
			FileName:     fh.Filename,
			MimeType:     mimeType,
		})
		if err != nil {
			service.DeleteMediaTempToken(db, tokRow)
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		if rel, err := service.CopyMessageMediaToPersistent(cfg.MediaPersistentDir, tokRow.FilePath, msgID, fh.Filename); err != nil {
			log.Warn("persist outbound media", zap.Error(err))
		} else if rel != "" {
			if err := db.Model(&model.Message{}).Where("id = ?", msgID).Update("stored_media_path", rel).Error; err != nil {
				log.Warn("update stored_media_path", zap.Error(err))
			}
		}
		service.DeleteMediaTempToken(db, tokRow)

		if err := service.PersistPortalOutboundWebhookMedia(db, inst.EvolutionInstanceName, sendRJ, "", sendKeyID, mediaKind, caption, fh.Filename); err != nil {
			log.Warn("portal outbound webhook media", zap.Error(err))
		}

		service.PublishInboxEvent(rdb, wid, map[string]interface{}{
			"type":            "message.created",
			"conversation_id": cid.String(),
		})

		return JSONSuccess(c, fiber.Map{"ok": true})
	}
}
