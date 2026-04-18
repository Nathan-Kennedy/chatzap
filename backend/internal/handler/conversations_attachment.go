package handler

import (
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
	"wa-saas/backend/internal/service"
)

func attachmentResponseContentType(messageType, mimeFromMsg, mimeHint string) string {
	ct := strings.TrimSpace(mimeHint)
	if ct == "" {
		ct = strings.TrimSpace(mimeFromMsg)
	}
	mt := strings.TrimSpace(strings.ToLower(messageType))
	// Não declarar OGG sem magic: o cliente faz sniff; inventar audio/ogg quebra o <audio>.
	if mt == "audio" && (ct == "" || strings.EqualFold(ct, "application/octet-stream")) {
		return "application/octet-stream"
	}
	if mt == "image" && (ct == "" || ct == "application/octet-stream") {
		return "image/jpeg"
	}
	if mt == "sticker" && (ct == "" || ct == "application/octet-stream") {
		return "image/webp"
	}
	if ct == "" {
		return "application/octet-stream"
	}
	return ct
}

// effectivePersistMime quando a Evolution devolve MIME vazio ou genérico — evita gravar .bin e força extensão correta.
func effectivePersistMime(hint string, msg *model.Message) string {
	h := strings.TrimSpace(hint)
	if h != "" && !strings.EqualFold(h, "application/octet-stream") {
		return h
	}
	if msg != nil {
		if m := strings.TrimSpace(msg.MimeType); m != "" && !strings.EqualFold(m, "application/octet-stream") {
			return m
		}
		mt := strings.TrimSpace(strings.ToLower(msg.MessageType))
		switch mt {
		case "audio":
			return "audio/ogg"
		case "image":
			return "image/jpeg"
		case "sticker":
			return "image/webp"
		case "video":
			return "video/mp4"
		}
	}
	if h != "" {
		return h
	}
	return ""
}

// persistInboundMedia grava bytes no disco na primeira visualização bem-sucedida (IA / cache offline).
func persistInboundMedia(db *gorm.DB, log *zap.Logger, cfg *config.Config, msg *model.Message, data []byte, mimeHint string) {
	if db == nil || cfg == nil || msg == nil || len(data) == 0 {
		return
	}
	if msg.Direction != "inbound" {
		return
	}
	if strings.TrimSpace(msg.MessageType) == "audio" && service.SniffAudioMIME(data) == "" {
		if log != nil {
			log.Debug("persist inbound audio skipped: no recognized container magic",
				zap.String("message_id", msg.ID.String()))
		}
		return
	}
	// Sobrescrever áudio em cache quando a Evolution devolve bytes com magic válido (cache antigo podia estar truncado).
	if strings.TrimSpace(msg.StoredMediaPath) != "" {
		if strings.TrimSpace(msg.MessageType) != "audio" || service.SniffAudioMIME(data) == "" {
			return
		}
	}
	mime := effectivePersistMime(mimeHint, msg)
	if mime == "" {
		mime = strings.TrimSpace(msg.MimeType)
	}
	rel, err := service.WriteMessageMediaBytes(cfg.MediaPersistentDir, msg.ID, data, msg.FileName, mime)
	if err != nil || rel == "" {
		log.Warn("persist inbound media", zap.Error(err))
		return
	}
	updates := map[string]interface{}{"stored_media_path": rel}
	if strings.TrimSpace(msg.MimeType) == "" && mime != "" {
		updates["mime_type"] = mime
	}
	if err := db.Model(&model.Message{}).Where("id = ? AND stored_media_path = ?", msg.ID, "").Updates(updates).Error; err != nil {
		log.Warn("update stored_media_path", zap.Error(err))
	}
}

func sanitizeAttachmentFilename(s string) string {
	s = filepath.Base(strings.TrimSpace(s))
	if s == "" || s == "." {
		return "attachment"
	}
	var b strings.Builder
	for _, r := range s {
		if r >= 32 && r <= 126 && r != '"' && r != '\\' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "attachment"
	}
	return out
}

func logAudioAttachmentDebug(log *zap.Logger, source string, messageID uuid.UUID, prefix []byte, sniff string) {
	if log == nil {
		return
	}
	n := len(prefix)
	if n > 16 {
		n = 16
	}
	hexP := ""
	if n > 0 {
		hexP = hex.EncodeToString(prefix[:n])
	}
	log.Debug("attachment audio",
		zap.String("source", source),
		zap.String("message_id", messageID.String()),
		zap.Int("read_len", len(prefix)),
		zap.String("sniff_mime", sniff),
		zap.String("hex_prefix", hexP),
	)
}

// HandleGetMessageAttachment GET /api/v1/conversations/:id/messages/:message_id/attachment — JWT; ficheiro local, proxy URL ou Evolution getBase64FromMediaMessage (mídia recebida).
func HandleGetMessageAttachment(log *zap.Logger, db *gorm.DB, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		cid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}
		mid, err := uuid.Parse(c.Params("message_id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "message_id inválido", nil)
		}

		var conv model.Conversation
		if err := db.Where("id = ? AND workspace_id = ?", cid, wid).First(&conv).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "conversa não encontrada", nil)
		}

		var msg model.Message
		if err := db.Where("id = ? AND conversation_id = ?", mid, cid).First(&msg).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "mensagem não encontrada", nil)
		}

		localKey := strings.TrimSpace(msg.StoredMediaPath)
		remote := strings.TrimSpace(msg.MediaRemoteURL)
		keyID := strings.TrimSpace(msg.ExternalID)
		waMsgJSON := strings.TrimSpace(msg.WaMediaMessageJSON)
		mt := strings.TrimSpace(msg.MessageType)
		if mt == "" {
			mt = "text"
		}
		canEvolutionGo := cfg.WhatsAppProvider == "evolution" && ev != nil && waMsgJSON != "" && mt != "text"
		canEvolution := cfg.WhatsAppProvider == "evolution" && ev != nil && keyID != "" && mt != "text"

		if localKey == "" && remote == "" && !canEvolution && !canEvolutionGo {
			return JSONError(c, fiber.StatusNotFound, "not_found", "sem anexo nesta mensagem", nil)
		}

		fn := sanitizeAttachmentFilename(msg.FileName)
		disposition := `inline; filename="` + fn + `"`

		// Áudio: servir primeiro pela Evolution (mesmo caminho que a transcrição). Cache local pode ter OggS no início mas ficheiro incompleto.
		if mt == "audio" && canEvolutionGo {
			var inst model.WhatsAppInstance
			if err := db.Where("id = ?", conv.WhatsAppInstanceID).First(&inst).Error; err != nil {
				log.Debug("attachment evolution audio-first: instância", zap.Error(err))
			} else {
				ctx, cancel := context.WithTimeout(c.Context(), 115*time.Second)
				defer cancel()
				data, mimeHint, err := ev.DownloadMediaEvolutionGo(ctx, inst.EvolutionInstanceToken, []byte(waMsgJSON))
				if err != nil {
					log.Warn("evolution downloadmedia (áudio preferencial)", zap.Error(err))
				} else if len(data) > 0 {
					s := service.SniffAudioMIME(data)
					logAudioAttachmentDebug(log, "evolution_go_audio_first", mid, data, s)
					if s == "" {
						log.Warn("evolution downloadmedia: áudio sem magic reconhecido",
							zap.String("message_id", mid.String()), zap.Int("bytes", len(data)))
						return JSONError(c, fiber.StatusBadGateway, "attachment_invalid_media",
							"bytes de áudio inválidos ou formato não suportado", nil)
					}
					mimeHint = s
					persistInboundMedia(db, log, cfg, &msg, data, mimeHint)
					c.Set("Content-Disposition", disposition)
					c.Set("Cache-Control", "private, max-age=300")
					ct := attachmentResponseContentType(mt, msg.MimeType, mimeHint)
					c.Set("Content-Type", ct)
					return c.Send(data)
				}
			}
		}

		clearInvalidLocalAudio := func() {
			if err := db.Model(&model.Message{}).
				Where("id = ? AND conversation_id = ?", mid, cid).
				Update("stored_media_path", "").Error; err != nil {
				log.Warn("clear invalid audio cache", zap.Error(err))
			} else {
				log.Debug("cleared stored_media_path (invalid audio magic)", zap.String("message_id", mid.String()))
			}
			localKey = ""
			msg.StoredMediaPath = ""
		}

		if localKey != "" {
			abs, err := service.ResolvePersistentMediaPath(cfg.MediaPersistentDir, localKey)
			if err != nil {
				log.Debug("attachment path", zap.Error(err))
				return JSONError(c, fiber.StatusNotFound, "not_found", "ficheiro não disponível", nil)
			}
			fi, statErr := os.Stat(abs)
			if statErr != nil {
				return JSONError(c, fiber.StatusNotFound, "not_found", "ficheiro não disponível", nil)
			}
			c.Set("Content-Disposition", disposition)
			c.Set("Cache-Control", "private, max-age=86400")
			const maxBuffered = 40 << 20 // evita ler vídeos grandes inteiros na RAM
			if fi.Size() <= maxBuffered {
				data, rerr := os.ReadFile(abs)
				if rerr != nil {
					log.Debug("attachment readfile", zap.Error(rerr))
					return JSONError(c, fiber.StatusNotFound, "not_found", "ficheiro não disponível", nil)
				}
				sniff := ""
				if mt == "audio" {
					sniff = service.SniffAudioMIME(data)
					logAudioAttachmentDebug(log, "local", mid, data, sniff)
					if sniff == "" {
						clearInvalidLocalAudio()
					}
				}
				if localKey != "" {
					ct := attachmentResponseContentType(mt, msg.MimeType, sniff)
					c.Set("Content-Type", ct)
					return c.Send(data)
				}
			} else {
				sniff := ""
				if mt == "audio" {
					head := make([]byte, 64)
					f, oerr := os.Open(abs)
					if oerr == nil {
						_, _ = f.Read(head)
						_ = f.Close()
						sniff = service.SniffAudioMIME(head)
						logAudioAttachmentDebug(log, "local_head", mid, head, sniff)
						if sniff == "" {
							clearInvalidLocalAudio()
						}
					}
				}
				if localKey != "" {
					ct := attachmentResponseContentType(mt, msg.MimeType, sniff)
					c.Set("Content-Type", ct)
					return c.SendFile(abs)
				}
			}
		}

		if remote != "" {
			u, perr := url.Parse(remote)
			if perr == nil && (u.Scheme == "https" || u.Scheme == "http") {
				client := &http.Client{Timeout: 90 * time.Second}
				req, rerr := http.NewRequestWithContext(c.Context(), http.MethodGet, remote, nil)
				if rerr == nil {
					req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
					resp, rerr := client.Do(req)
					if rerr == nil {
						if resp.StatusCode < 400 {
							ct := strings.TrimSpace(resp.Header.Get("Content-Type"))
							if ct == "" || ct == "application/octet-stream" {
								if m := strings.TrimSpace(msg.MimeType); m != "" {
									ct = m
								} else {
									ct = "application/octet-stream"
								}
							}
							c.Set("Content-Type", ct)
							c.Set("Content-Disposition", disposition)
							c.Set("Cache-Control", "private, max-age=300")
							c.Status(fiber.StatusOK)
							defer resp.Body.Close()
							_, err := io.Copy(c.Response().BodyWriter(), resp.Body)
							return err
						}
						_ = resp.Body.Close()
					} else {
						log.Debug("attachment proxy http", zap.Error(rerr))
					}
				}
			}
			log.Debug("attachment proxy falhou ou URL inválida; a tentar Evolution", zap.String("remote", truncateForLog(remote)))
		}

		if (canEvolutionGo && mt != "audio") || canEvolution {
			var inst model.WhatsAppInstance
			if err := db.Where("id = ?", conv.WhatsAppInstanceID).First(&inst).Error; err != nil {
				log.Debug("attachment evolution: instância", zap.Error(err))
			} else {
				if canEvolutionGo && mt != "audio" {
					ctx, cancel := context.WithTimeout(c.Context(), 115*time.Second)
					defer cancel()
					data, mimeHint, err := ev.DownloadMediaEvolutionGo(ctx, inst.EvolutionInstanceToken, []byte(waMsgJSON))
					if err != nil {
						log.Warn("evolution downloadmedia", zap.Error(err))
					} else if len(data) > 0 {
						persistInboundMedia(db, log, cfg, &msg, data, mimeHint)
						ct := attachmentResponseContentType(mt, msg.MimeType, mimeHint)
						c.Set("Content-Type", ct)
						c.Set("Content-Disposition", disposition)
						c.Set("Cache-Control", "private, max-age=300")
						return c.Send(data)
					}
				}
				if canEvolution {
					fromMe := msg.Direction == "outbound"
					ctx, cancel := context.WithTimeout(c.Context(), 115*time.Second)
					defer cancel()
					data, mimeHint, err := ev.GetBase64FromMediaMessage(ctx, inst.EvolutionInstanceName, inst.EvolutionInstanceToken, keyID, conv.ContactJID, fromMe, mt == "video")
					if err != nil {
						log.Warn("evolution getBase64FromMediaMessage", zap.String("key_id", keyID), zap.Error(err))
					} else if len(data) > 0 {
						if mt == "audio" {
							s := service.SniffAudioMIME(data)
							logAudioAttachmentDebug(log, "evolution_base64", mid, data, s)
							if s == "" {
								log.Warn("evolution getBase64: áudio sem magic reconhecido",
									zap.String("message_id", mid.String()), zap.Int("bytes", len(data)))
								return JSONError(c, fiber.StatusBadGateway, "attachment_invalid_media",
									"bytes de áudio inválidos ou formato não suportado", nil)
							}
							mimeHint = s
						}
						persistInboundMedia(db, log, cfg, &msg, data, mimeHint)
						ct := attachmentResponseContentType(mt, msg.MimeType, mimeHint)
						c.Set("Content-Type", ct)
						c.Set("Content-Disposition", disposition)
						c.Set("Cache-Control", "private, max-age=300")
						return c.Send(data)
					}
				}
			}
		}

		return JSONError(c, fiber.StatusNotFound, "not_found", "não foi possível obter o anexo (URL expirada ou Evolution indisponível)", nil)
	}
}

func truncateForLog(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 80 {
		return s
	}
	return s[:80] + "…"
}
