package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/model"
)

// SendAutoReplyVoice sintetiza voz por segmentos (como bolhas de texto), publica URL temporária e envia áudio via Evolution.
func SendAutoReplyVoice(
	ctx context.Context,
	log *zap.Logger,
	db *gorm.DB,
	rdb *redis.Client,
	cfg *config.Config,
	ev *EvolutionClient,
	appEncKey string,
	agent *model.AIAgent,
	replyText string,
	conversationID uuid.UUID,
	workspaceID uuid.UUID,
	evInstanceToken string,
	evolutionInstanceSlug string,
	contactJID string,
) error {
	if cfg == nil || ev == nil || agent == nil {
		return fmt.Errorf("deps em falta")
	}

	parts := voiceReplyChunks(replyText)
	if len(parts) == 0 {
		return fmt.Errorf("texto vazio para TTS")
	}

	prov := NormalizeTTSProvider(agent.TTSProvider)
	voice := EffectiveOpenAITTSVoice(agent, cfg)

	var openAIKey string
	if prov == TTSProviderOpenAI {
		key, kerr := ResolveOpenAITTSAPIKey(appEncKey, agent)
		if kerr != nil {
			return kerr
		}
		openAIKey = key
	}

	var omniBase string
	var omniOpts *OmnivoiceSpeechOpts
	if prov == TTSProviderOmnivoice {
		def := ""
		if cfg != nil {
			def = cfg.OmnivoiceDefaultBaseURL
		}
		omniBase = ResolveOmnivoiceBaseURL(agent, def)
		if omniBase == "" {
			return fmt.Errorf("omnivoice_base_url em falta (agente ou OMNIVOICE_DEFAULT_BASE_URL)")
		}
		omniOpts = OmnivoiceAutoReplyOptsFromConfig(cfg)
		if log != nil {
			mapped := mapVoiceForOmnivoiceWithOpts(voice, omniOpts)
			log.Info("auto-reply: OmniVoice TTS",
				zap.String("omnivoice_base_url", omniBase),
				zap.String("openai_tts_voice", voice),
				zap.String("omnivoice_voice", mapped),
				zap.Int("chunks", len(parts)))
		}
	}

	var elevenKey string
	if prov == TTSProviderElevenLabs {
		k, err := ResolveElevenLabsAPIKey(appEncKey, cfg, agent)
		if err != nil {
			return err
		}
		elevenKey = k
	}

	var kokoroBase string
	if prov == TTSProviderKokoro {
		def := ""
		if cfg != nil {
			def = cfg.KokoroDefaultBaseURL
		}
		kokoroBase = ResolveKokoroBaseURL(agent, def)
		if kokoroBase == "" {
			return fmt.Errorf("kokoro_base_url em falta (agente ou KOKORO_DEFAULT_BASE_URL)")
		}
	}

	mimeType := "audio/mpeg"
	ext := ".mp3"
	if prov == TTSProviderOmnivoice {
		mimeType = "audio/wav"
		ext = ".wav"
	}
	if prov == TTSProviderKokoro {
		mimeType = "audio/wav"
		ext = ".wav"
	}
	ttlMin := cfg.MediaTokenTTLMinutes
	if ttlMin <= 0 {
		ttlMin = 30
	}
	ttl := time.Duration(ttlMin) * time.Minute

	for i, part := range parts {
		if i > 0 {
			pause := PauseBetweenChunks()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(pause):
			}
		}

		text := TruncateForTTS(strings.TrimSpace(part), MaxOpenAITTSInputRunes)
		if text == "" {
			continue
		}

		var audio []byte
		var err error
		switch prov {
		case TTSProviderOpenAI:
			inst := OpenAITTSInstructionsPtr(cfg)
			audio, err = SynthOpenAITTS(ctx, openAIKey, EffectiveOpenAITTSModel(agent, cfg), voice, text, inst)
		case TTSProviderOmnivoice:
			audio, err = SynthOmnivoiceOpenAICompat(ctx, omniBase, "", OmnivoiceClientModel, voice, text, omniOpts)
			if err != nil && log != nil {
				log.Warn("auto-reply: TTS OmniVoice falhou", zap.String("omnivoice_base_url", omniBase), zap.Int("chunk_index", i), zap.Error(err))
			}
		case TTSProviderElevenLabs:
			audio, err = SynthElevenLabs(ctx, elevenKey, voice, text, EffectiveElevenLabsModel(cfg))
		case TTSProviderKokoro:
			audio, err = SynthKokoroOpenAICompat(ctx, kokoroBase, "", EffectiveKokoroTTSModel(cfg), voice, text)
		default:
			return fmt.Errorf("tts_provider inválido: %s", prov)
		}
		if err != nil {
			return fmt.Errorf("voice chunk %d: %w", i+1, err)
		}
		if len(audio) == 0 {
			return fmt.Errorf("voice chunk %d: TTS devolveu bytes vazios", i+1)
		}

		filename := fmt.Sprintf("resposta-%d%s", i+1, ext)
		plainToken, tokRow, err := NewMediaTempToken(db, cfg.MediaUploadDir, ttl, filename, mimeType, bytes.NewReader(audio), cfg.MediaMaxUploadBytes)
		if err != nil {
			return fmt.Errorf("media temp chunk %d: %w", i+1, err)
		}
		publicURL := cfg.MediaTempFetchURL(plainToken)
		if publicURL == "" {
			DeleteMediaTempToken(db, tokRow)
			return fmt.Errorf("PUBLIC_MEDIA_BASE_URL em falta")
		}

		sendRaw, err := ev.SendMedia(ctx, evInstanceToken, contactJID, "audio", publicURL, "", filename)
		if err != nil {
			DeleteMediaTempToken(db, tokRow)
			return fmt.Errorf("send media chunk %d: %w", i+1, err)
		}
		sendRJ, sendKeyID := ParseEvolutionSendTextResponse(sendRaw)
		if strings.TrimSpace(sendRJ) == "" {
			sendRJ = contactJID
		}

		msgID, err := RecordOutbound(db, conversationID, OutboundRecord{
			Body:        text,
			ExternalID:  sendKeyID,
			MessageType: "audio",
			FileName:    filename,
			MimeType:    mimeType,
		})
		if err != nil {
			DeleteMediaTempToken(db, tokRow)
			return fmt.Errorf("record outbound chunk %d: %w", i+1, err)
		}
		if rel, err := CopyMessageMediaToPersistent(cfg.MediaPersistentDir, tokRow.FilePath, msgID, filename); err != nil {
			if log != nil {
				log.Warn("persist auto-reply voice", zap.Error(err))
			}
		} else if rel != "" {
			_ = db.Model(&model.Message{}).Where("id = ?", msgID).Update("stored_media_path", rel).Error
		}
		DeleteMediaTempToken(db, tokRow)

		if err := PersistPortalOutboundWebhookMedia(db, evolutionInstanceSlug, sendRJ, "", sendKeyID, "audio", "", filename); err != nil && log != nil {
			log.Debug("portal outbound webhook media", zap.Error(err))
		}
		if workspaceID != uuid.Nil && rdb != nil {
			PublishInboxEvent(rdb, workspaceID, map[string]interface{}{
				"type":            "message.created",
				"conversation_id": conversationID.String(),
			})
		}
	}

	return nil
}

// voiceReplyChunks reutiliza a mesma divisão que as bolhas de texto (parágrafos / limite de runas).
func voiceReplyChunks(reply string) []string {
	raw := SplitReplyIntoMessageChunks(strings.TrimSpace(reply), 0)
	var out []string
	for _, c := range raw {
		c = strings.TrimSpace(c)
		if c != "" {
			out = append(out, c)
		}
	}
	return out
}
