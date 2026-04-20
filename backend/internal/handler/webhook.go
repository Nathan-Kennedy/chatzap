package handler

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/model"
	"wa-saas/backend/internal/service"
)

type WebhookDeps struct {
	Log       *zap.Logger
	DB        *gorm.DB
	Redis     *redis.Client
	Cfg       *config.Config
	Evolution *service.EvolutionClient
	LLM       service.LLM
}

// HandleWhatsAppWebhook POST /webhooks/whatsapp/:instance_id (playbook).
func HandleWhatsAppWebhook(d WebhookDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		instanceParam := c.Params("instance_id")
		if d.Cfg.AllowedInstanceIDs != nil {
			if _, ok := d.Cfg.AllowedInstanceIDs[instanceParam]; !ok {
				return JSONError(c, fiber.StatusForbidden, "instance_not_allowed", "instância não autorizada nesta API", nil)
			}
		}

		raw := append([]byte(nil), c.Body()...)
		var payload service.EvolutionWebhookPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_json", "corpo JSON inválido", nil)
		}

		instance := payload.Instance
		if instance == "" {
			instance = instanceParam
		}
		if instance != instanceParam {
			d.Log.Warn("webhook instance mismatch",
				zap.String("path_instance", instanceParam),
				zap.String("body_instance", payload.Instance),
			)
		}

		rec := model.WebhookMessage{
			InstanceID: instanceParam,
			Event:      payload.Event,
			RawPayload: raw,
		}

		dataForParse := service.NormalizeWebhookData(payload.Data)
		inbound, ok := service.ParseInboundFromEvolution(payload.Event, dataForParse)
		canonical := service.InboundCanonicalJID(inbound.From, inbound.RemoteJidAlt)
		if ok && canonical != "" && inbound.Text != "" {
			rec.RemoteJID = canonical
			rec.Body = inbound.Text
			if inbound.FromMe {
				rec.Direction = "outbound"
			} else {
				rec.Direction = "inbound"
			}
		} else {
			rec.Direction = "event"
		}

		if err := d.DB.Create(&rec).Error; err != nil {
			d.Log.Error("persist webhook", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", "falha ao gravar evento", nil)
		}

		// Path :instance_id é a fonte de verdade (igual ao URL registado em sync-webhook).
		// O corpo pode trazer `instance` como UUID; usar isso quebrava o match com evolution_instance_name.
		evoName := strings.ToLower(strings.TrimSpace(instanceParam))
		var workspaceID uuid.UUID
		var conversationID uuid.UUID
		var inboundMessageID uuid.UUID
		if ok && canonical != "" && inbound.Text != "" && !inbound.FromMe {
			wid, cid, mid, err := service.UpsertInboundMessage(d.DB, d.Log, evoName, inbound)
			if err != nil {
				d.Log.Error("inbox upsert", zap.Error(err))
			} else if wid != uuid.Nil && cid != uuid.Nil {
				workspaceID = wid
				conversationID = cid
				inboundMessageID = mid
				service.PublishInboxEvent(d.Redis, wid, map[string]interface{}{
					"type":            "message.created",
					"payload":         map[string]string{"conversation_id": cid.String()},
					"conversation_id": cid.String(),
				})
			} else {
				d.Log.Warn("webhook: inbound parseado mas não gravado na inbox",
					zap.String("path_instance", instanceParam),
					zap.String("event", payload.Event),
					zap.String("canonical_jid", canonical),
				)
			}
		} else if ok && canonical != "" && inbound.Text != "" && inbound.FromMe {
			wid, cid, err := service.UpsertOutboundFromWebhook(d.DB, d.Log, evoName, inbound)
			if err != nil {
				d.Log.Error("inbox outbound webhook", zap.Error(err))
			} else if wid != uuid.Nil && cid != uuid.Nil {
				workspaceID = wid
				conversationID = cid
				service.PublishInboxEvent(d.Redis, wid, map[string]interface{}{
					"type":            "message.created",
					"payload":         map[string]string{"conversation_id": cid.String()},
					"conversation_id": cid.String(),
				})
			}
		} else if service.IsInboundMessageEvent(payload.Event) && !ok {
			d.Log.Warn("webhook: evento de mensagem sem texto/canonical parseável",
				zap.String("path_instance", instanceParam),
				zap.String("event", payload.Event),
				zap.Bool("parse_ok", ok),
				zap.String("canonical_jid", canonical),
			)
		}

		llm := d.LLM
		if workspaceID != uuid.Nil && strings.TrimSpace(d.Cfg.AppEncryptionKey) != "" {
			if waLLM, err := service.WorkspaceAutoReplyLLM(d.DB, d.Cfg.AppEncryptionKey, workspaceID); err != nil {
				d.Log.Warn("auto-reply: resolver agente workspace", zap.Error(err))
			} else if waLLM != nil {
				llm = waLLM
			}
		}

		if d.Cfg.AutoReplyEnabled && llm != nil && ok && inbound.Text != "" && !inbound.FromMe && conversationID != uuid.Nil {
			from := canonical
			evInstanceToken := d.Cfg.EvolutionInstanceName
			if payload.Instance != "" {
				evInstanceToken = payload.Instance
			}
			if evInstanceToken == "" {
				evInstanceToken = instanceParam
			}
			var inst model.WhatsAppInstance
			evoKey := strings.ToLower(strings.TrimSpace(instanceParam))
			if err := d.DB.Where("evolution_instance_name = ?", evoKey).First(&inst).Error; err == nil {
				if strings.TrimSpace(inst.EvolutionInstanceToken) != "" {
					evInstanceToken = inst.EvolutionInstanceToken
				}
			}
			cid := conversationID
			wid := workspaceID
			evoSlug := instanceParam
			inCopy := inbound
			curInboundID := inboundMessageID
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
				defer cancel()
				text := inCopy.Text
				if d.Evolution != nil && strings.TrimSpace(inCopy.WaMediaMessageJSON) != "" && strings.TrimSpace(inCopy.MessageType) == "audio" {
					audioCtx, audioCancel := context.WithTimeout(ctx, 3*time.Minute)
					audioBytes, mimeH, aerr := d.Evolution.DownloadMediaEvolutionGo(audioCtx, evInstanceToken, []byte(inCopy.WaMediaMessageJSON))
					audioCancel()
					if aerr != nil || len(audioBytes) == 0 {
						d.Log.Warn("auto-reply: download áudio Evolution Go", zap.Error(aerr))
					} else {
						if s := service.SniffAudioMIME(audioBytes); s != "" {
							mimeH = s
						}
						tr, terr := service.TranscribeVoiceNoteWithConfig(ctx, llm, d.Cfg, audioBytes, mimeH)
						if terr != nil {
							d.Log.Warn("auto-reply: transcrição de voz", zap.Error(terr))
						} else if t := strings.TrimSpace(tr); t != "" {
							text = "[Mensagem de voz] " + t
							kid := strings.TrimSpace(inCopy.KeyID)
							if kid != "" {
								_ = d.DB.Model(&model.Message{}).
									Where("conversation_id = ? AND external_id = ?", cid, kid).
									Update("body", text).Error
							} else {
								since := time.Now().UTC().Add(-3 * time.Minute)
								var row model.Message
								if err := d.DB.Where("conversation_id = ? AND direction = ? AND message_type = ? AND created_at >= ?", cid, "inbound", "audio", since).
									Order("created_at DESC").
									First(&row).Error; err == nil && row.ID != uuid.Nil {
									_ = d.DB.Model(&model.Message{}).Where("id = ?", row.ID).Update("body", text).Error
								}
							}
						}
					}
				}
				hint := service.ContinuationStyleHint(d.DB, cid)
				suffix := "\n---\nNova mensagem do cliente (responde em continuação natural, usando o contexto acima):\n" + text
				if strings.HasPrefix(strings.TrimSpace(text), "[Mensagem de voz]") {
					suffix = "\n---\nO cliente enviou uma mensagem de voz. Usa a transcrição abaixo como o que ele disse (não digas que não consegues ouvir áudio):\n" + text
				}
				userForLLM := text
				hist, histErr := service.BuildWhatsAppHistoryForLLM(d.DB, cid, text, service.DefaultHistoryMaxMessages, service.DefaultHistoryMaxRunes, curInboundID)
				if histErr != nil {
					d.Log.Debug("auto-reply: histórico conversa", zap.Error(histErr))
				} else if strings.TrimSpace(hist) != "" {
					userForLLM = hist + hint + suffix
				} else if strings.TrimSpace(hint) != "" {
					userForLLM = hint + suffix
				}
				reply, err := llm.Reply(ctx, userForLLM)
				if err != nil {
					d.Log.Error("llm reply", zap.String("provider", service.ProviderName(llm)), zap.Error(err))
					return
				}
				reply = service.SanitizeLLMTextForWhatsApp(reply)
				if strings.TrimSpace(hint) != "" {
					reply = service.StripLeadingSalutationNameLine(reply)
				}
				if d.Evolution == nil {
					d.Log.Warn("auto-reply: resposta LLM não enviada por WhatsApp (Evolution não configurado)",
						zap.String("provider", d.Cfg.WhatsAppProvider),
					)
					return
				}
				var agentRow *model.AIAgent
				if wid != uuid.Nil && strings.TrimSpace(d.Cfg.AppEncryptionKey) != "" {
					if a, aerr := service.WorkspaceAutoReplyAgent(d.DB, wid); aerr != nil {
						d.Log.Debug("auto-reply: carregar agente", zap.Error(aerr))
					} else {
						agentRow = a
					}
				}
				if agentRow != nil && agentRow.VoiceReplyEnabled &&
					service.NormalizeTTSProvider(agentRow.TTSProvider) != service.TTSProviderNone {
					if err := service.SendAutoReplyVoice(ctx, d.Log, d.DB, d.Redis, d.Cfg, d.Evolution,
						d.Cfg.AppEncryptionKey, agentRow, reply, cid, wid, evInstanceToken, evoSlug, from); err != nil {
						d.Log.Warn("auto-reply: envio voz falhou — a enviar resposta em texto",
							zap.Error(err))
					} else {
						return
					}
				}
				chunks := service.SplitReplyIntoMessageChunks(reply, 0)
				if len(chunks) == 0 {
					return
				}
				firstBubble := true
				for i, part := range chunks {
					part = strings.TrimSpace(part)
					if part == "" {
						continue
					}
					if !firstBubble {
						pause := service.PauseBetweenChunks()
						select {
						case <-ctx.Done():
							return
						case <-time.After(pause):
						}
					}
					typing := service.TypingDelayBeforeChunk(part, firstBubble)
					typingMs := int(typing / time.Millisecond)
					if typingMs < 1 {
						typingMs = 1
					}
					if err := d.Evolution.SendPresence(ctx, evoSlug, evInstanceToken, from, "composing", typingMs); err != nil {
						d.Log.Warn("auto-reply: sendPresence (digitando) falhou — Evolution Go usa POST /message/presence; Node usa POST /chat/sendPresence/{instance}",
							zap.Error(err))
					}
					select {
					case <-ctx.Done():
						return
					case <-time.After(typing):
					}
					sendRaw, err := d.Evolution.SendText(ctx, evInstanceToken, from, part)
					if err != nil {
						d.Log.Error("evolution send", zap.Int("chunk_index", i), zap.Error(err))
						return
					}
					_, sendKeyID := service.ParseEvolutionSendTextResponse(sendRaw)
					if err := service.RecordOutboundMessage(d.DB, cid, part, sendKeyID); err != nil {
						d.Log.Error("record outbound", zap.Error(err))
					}
					if err := service.PersistPortalOutboundWebhook(d.DB, evoSlug, from, "", part, sendKeyID); err != nil {
						d.Log.Debug("portal outbound webhook audit", zap.Error(err))
					}
					if wid != uuid.Nil {
						service.PublishInboxEvent(d.Redis, wid, map[string]interface{}{
							"type":            "message.created",
							"conversation_id": cid.String(),
						})
					}
					firstBubble = false
				}
			}()
		} else if d.Cfg.AutoReplyEnabled && llm == nil && ok && inbound.Text != "" && !inbound.FromMe && conversationID != uuid.Nil {
			d.Log.Warn("auto-reply: sem LLM — inbox actualizada mas nenhuma resposta gerada. Defina GEMINI_API_KEY ou OPENAI_API_KEY no Coolify, ou guarde a API key no agente com «usar no WhatsApp» activo",
				zap.String("instance", instanceParam),
				zap.String("conversation_id", conversationID.String()),
			)
		}

		return JSONSuccess(c, fiber.Map{
			"received":   true,
			"event":      payload.Event,
			"message_id": rec.ID.String(),
		})
	}
}
