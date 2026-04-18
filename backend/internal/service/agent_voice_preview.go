package service

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/model"
)

// AgentVoicePreviewPhrase texto fixo para amostra de voz (~10s conforme motor TTS).
func AgentVoicePreviewPhrase(agentName string) string {
	n := strings.TrimSpace(agentName)
	n = strings.ReplaceAll(n, "\n", " ")
	n = strings.ReplaceAll(n, "\r", "")
	if n == "" {
		n = "assistente"
	}
	return fmt.Sprintf("Olá, sou a agente %s. Como posso te ajudar hoje?", n)
}

// RegenerateAgentVoicePreview gera áudio TTS e grava em MEDIA_PERSISTENT_DIR/agent_voice_previews/.
// Limpa voice_preview_path quando voz desligada ou falha (regista warn).
func RegenerateAgentVoicePreview(ctx context.Context, log *zap.Logger, db *gorm.DB, cfg *config.Config, appEncKey string, agent *model.AIAgent) error {
	if cfg == nil || agent == nil || db == nil {
		return fmt.Errorf("deps em falta")
	}
	persist := strings.TrimSpace(cfg.MediaPersistentDir)
	if persist == "" {
		return fmt.Errorf("MEDIA_PERSISTENT_DIR vazio")
	}

	prov := NormalizeTTSProvider(agent.TTSProvider)
	if !agent.VoiceReplyEnabled || prov == TTSProviderNone {
		_ = removeVoicePreviewFile(cfg, agent.VoicePreviewPath)
		return db.Model(agent).Update("voice_preview_path", "").Error
	}

	text := TruncateForTTS(AgentVoicePreviewPhrase(agent.Name), MaxOpenAITTSInputRunes)
	if text == "" {
		return fmt.Errorf("frase de prévia vazia")
	}

	voice := EffectiveOpenAITTSVoice(agent, cfg)
	var audio []byte
	var ext string
	var err error

	switch prov {
	case TTSProviderOpenAI:
		key, kerr := ResolveOpenAITTSAPIKey(appEncKey, agent)
		if kerr != nil {
			if log != nil {
				log.Warn("prévia voz: sem chave OpenAI", zap.String("agent_id", agent.ID.String()), zap.Error(kerr))
			}
			_ = removeVoicePreviewFile(cfg, agent.VoicePreviewPath)
			return db.Model(agent).Update("voice_preview_path", "").Error
		}
		inst := OpenAITTSInstructionsPtr(cfg)
		audio, err = SynthOpenAITTS(ctx, key, EffectiveOpenAITTSModel(agent, cfg), voice, text, inst)
		ext = ".mp3"
	case TTSProviderOmnivoice:
		def := ""
		if cfg != nil {
			def = cfg.OmnivoiceDefaultBaseURL
		}
		omniBase := ResolveOmnivoiceBaseURL(agent, def)
		if omniBase == "" {
			if log != nil {
				log.Warn("prévia voz: omnivoice_base_url em falta", zap.String("agent_id", agent.ID.String()))
			}
			_ = removeVoicePreviewFile(cfg, agent.VoicePreviewPath)
			return db.Model(agent).Update("voice_preview_path", "").Error
		}
		opts := OmnivoiceAutoReplyOptsFromConfig(cfg)
		audio, err = SynthOmnivoiceOpenAICompat(ctx, omniBase, "", OmnivoiceClientModel, voice, text, opts)
		ext = ".wav"
	case TTSProviderElevenLabs:
		key, kerr := ResolveElevenLabsAPIKey(appEncKey, cfg, agent)
		if kerr != nil {
			if log != nil {
				log.Warn("prévia voz: sem chave ElevenLabs", zap.String("agent_id", agent.ID.String()), zap.Error(kerr))
			}
			_ = removeVoicePreviewFile(cfg, agent.VoicePreviewPath)
			return db.Model(agent).Update("voice_preview_path", "").Error
		}
		audio, err = SynthElevenLabs(ctx, key, voice, text, EffectiveElevenLabsModel(cfg))
		ext = ".mp3"
	case TTSProviderKokoro:
		def := ""
		if cfg != nil {
			def = cfg.KokoroDefaultBaseURL
		}
		kokoroBase := ResolveKokoroBaseURL(agent, def)
		if kokoroBase == "" {
			if log != nil {
				log.Warn("prévia voz: kokoro_base_url em falta", zap.String("agent_id", agent.ID.String()))
			}
			_ = removeVoicePreviewFile(cfg, agent.VoicePreviewPath)
			return db.Model(agent).Update("voice_preview_path", "").Error
		}
		audio, err = SynthKokoroOpenAICompat(ctx, kokoroBase, "", EffectiveKokoroTTSModel(cfg), voice, text)
		ext = ".wav"
	default:
		return fmt.Errorf("tts_provider inválido: %s", prov)
	}
	if err != nil {
		if log != nil {
			log.Warn("prévia voz: síntese falhou", zap.String("agent_id", agent.ID.String()), zap.Error(err))
		}
		_ = removeVoicePreviewFile(cfg, agent.VoicePreviewPath)
		return db.Model(agent).Update("voice_preview_path", "").Error
	}
	if len(audio) == 0 {
		if log != nil {
			log.Warn("prévia voz: bytes vazios", zap.String("agent_id", agent.ID.String()))
		}
		_ = removeVoicePreviewFile(cfg, agent.VoicePreviewPath)
		return db.Model(agent).Update("voice_preview_path", "").Error
	}

	sub := path.Join("agent_voice_previews", agent.ID.String()+ext)
	full, err := filepath.Abs(filepath.Join(persist, filepath.FromSlash(sub)))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return err
	}
	oldRel := strings.TrimSpace(agent.VoicePreviewPath)
	if oldRel != "" && oldRel != sub {
		_ = removeVoicePreviewFile(cfg, oldRel)
	}
	if err := os.WriteFile(full, audio, 0o640); err != nil {
		return err
	}
	return db.Model(agent).Update("voice_preview_path", sub).Error
}

// RemoveStoredVoicePreview apaga o ficheiro de prévia (ex.: ao eliminar agente).
func RemoveStoredVoicePreview(cfg *config.Config, rel string) error {
	return removeVoicePreviewFile(cfg, rel)
}

func removeVoicePreviewFile(cfg *config.Config, rel string) error {
	rel = strings.TrimSpace(rel)
	if rel == "" || cfg == nil {
		return nil
	}
	abs, err := ResolvePersistentMediaPath(cfg.MediaPersistentDir, rel)
	if err != nil {
		return nil
	}
	return os.Remove(abs)
}

// VoicePreviewNeedsRegenerate indica se o PATCH alterou dados que invalidam a prévia.
func VoicePreviewNeedsRegenerate(updates map[string]interface{}) bool {
	if len(updates) == 0 {
		return false
	}
	for _, k := range []string{
		"name",
		"voice_reply_enabled",
		"tts_provider",
		"openai_tts_voice",
		"openai_tts_model",
		"omnivoice_base_url",
		"kokoro_base_url",
		"openai_tts_api_key_cipher",
		"elevenlabs_api_key_cipher",
	} {
		if _, ok := updates[k]; ok {
			return true
		}
	}
	return false
}
