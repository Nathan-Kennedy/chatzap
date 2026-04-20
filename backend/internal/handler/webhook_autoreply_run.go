package handler

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"wa-saas/backend/internal/model"
	"wa-saas/backend/internal/service"
)

// runWhatsAppAutoReply gera e envia a resposta automática para um lote de mensagens inbound (debounce).
func runWhatsAppAutoReply(
	ctx context.Context,
	d WebhookDeps,
	llm service.LLM,
	cid, wid uuid.UUID,
	evoSlug, evInstanceToken, from string,
	batch []service.AutoReplyQueueItem,
) error {
	if len(batch) == 0 {
		return nil
	}

	lines := make([]string, 0, len(batch))
	excludeIDs := make([]uuid.UUID, 0, len(batch))
	for _, it := range batch {
		if it.MessageID != uuid.Nil {
			excludeIDs = append(excludeIDs, it.MessageID)
		}
		text := it.Text
		if d.Evolution != nil && strings.TrimSpace(it.WaMediaMessageJSON) != "" && strings.TrimSpace(it.MessageType) == "audio" {
			audioCtx, audioCancel := context.WithTimeout(ctx, 3*time.Minute)
			audioBytes, mimeH, aerr := d.Evolution.DownloadMediaEvolutionGo(audioCtx, evInstanceToken, []byte(it.WaMediaMessageJSON))
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
					kid := strings.TrimSpace(it.KeyID)
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
		lines = append(lines, text)
	}

	combined := strings.Join(lines, "\n\n")
	hint := service.ContinuationStyleHint(d.DB, cid)

	var suffix string
	if len(batch) == 1 {
		suffix = "\n---\nNova mensagem do cliente (responde em continuação natural, usando o contexto acima):\n" + combined
		if strings.HasPrefix(strings.TrimSpace(combined), "[Mensagem de voz]") {
			suffix = "\n---\nO cliente enviou uma mensagem de voz. Usa a transcrição abaixo como o que ele disse (não digas que não consegues ouvir áudio):\n" + combined
		}
	} else {
		suffix = "\n---\nO cliente enviou várias mensagens seguidas. Lê tudo como um único pedido e responde de uma vez, de forma natural:\n\n" + combined
	}

	userForLLM := combined
	hist, histErr := service.BuildWhatsAppHistoryForLLM(d.DB, cid, combined, service.DefaultHistoryMaxMessages, service.DefaultHistoryMaxRunes, excludeIDs)
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
		return err
	}
	reply = service.SanitizeLLMTextForWhatsApp(reply)
	if strings.TrimSpace(hint) != "" {
		reply = service.StripLeadingSalutationNameLine(reply)
	}
	if d.Evolution == nil {
		d.Log.Warn("auto-reply: resposta LLM não enviada por WhatsApp (Evolution não configurado)",
			zap.String("provider", d.Cfg.WhatsAppProvider),
		)
		return nil
	}

	var agentRow *model.AIAgent
	if wid != uuid.Nil {
		if a, aerr := service.WorkspaceAutoReplyAgent(d.DB, wid); aerr != nil {
			d.Log.Debug("auto-reply: carregar agente", zap.Error(aerr))
		} else {
			agentRow = a
		}
	}
	if agentRow != nil && agentRow.VoiceReplyEnabled &&
		service.NormalizeTTSProvider(agentRow.TTSProvider) != service.TTSProviderNone {
		d.Log.Warn("auto-reply: tentativa de resposta em voz",
			zap.String("tts_provider", agentRow.TTSProvider),
			zap.Bool("has_app_encryption_key", strings.TrimSpace(d.Cfg.AppEncryptionKey) != ""),
		)
		tryVoice := service.PreferVoiceForAutoReply(reply)
		if tryVoice {
			if err := service.SendAutoReplyVoice(ctx, d.Log, d.DB, d.Redis, d.Cfg, d.Evolution,
				d.Cfg.AppEncryptionKey, agentRow, reply, cid, wid, evInstanceToken, evoSlug, from); err != nil {
				d.Log.Warn("auto-reply: envio voz falhou — a enviar resposta em texto",
					zap.Error(err))
			} else {
				if service.ReplyLooksGravablePT(reply) {
					gap := time.Duration(0)
					if d.Cfg != nil && d.Cfg.VoiceToTextGapMs > 0 {
						gap = time.Duration(d.Cfg.VoiceToTextGapMs) * time.Millisecond
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(gap):
					}
					follow, ferr := service.GravableFollowUpText(ctx, d.Cfg, reply)
					if ferr != nil {
						d.Log.Warn("auto-reply: resumo pós-voz", zap.Error(ferr))
					}
					if strings.TrimSpace(follow) != "" {
						if err := sendWhatsAppAutoReplyChunks(ctx, d, cid, wid, evoSlug, evInstanceToken, from, follow); err != nil {
							d.Log.Warn("auto-reply: texto após voz (gravável)", zap.Error(err))
						}
					}
				}
				return nil
			}
		} else {
			d.Log.Warn("auto-reply: resposta curta — canal texto (TTS reserva-se para mensagens mais longas)",
				zap.Int("runes", utf8.RuneCountInString(strings.TrimSpace(reply))),
			)
		}
	} else if agentRow != nil {
		ttsN := service.NormalizeTTSProvider(agentRow.TTSProvider)
		switch {
		case !agentRow.VoiceReplyEnabled && ttsN != service.TTSProviderNone:
			d.Log.Warn("auto-reply: só texto — na BD há tts_provider mas voice_reply_enabled=false; ligue «Responder em áudio (TTS)» e guarde",
				zap.String("tts_provider", agentRow.TTSProvider),
				zap.String("agent_id", agentRow.ID.String()))
		case agentRow.VoiceReplyEnabled && ttsN == service.TTSProviderNone:
			d.Log.Warn("auto-reply: só texto — voice_reply_enabled=true mas tts_provider inválido/none; escolha Gemini TTS e guarde",
				zap.String("agent_id", agentRow.ID.String()))
		}
	}
	return sendWhatsAppAutoReplyChunks(ctx, d, cid, wid, evoSlug, evInstanceToken, from, reply)
}
