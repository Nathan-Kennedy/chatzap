package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AIAgent configuração de LLM por workspace (chave encriptada em repouso).
type AIAgent struct {
	ID                      uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID             uuid.UUID `gorm:"type:uuid;index;not null"`
	Name                    string    `gorm:"size:128;not null"`
	Provider                string    `gorm:"size:16;not null"` // gemini | openai
	Model                   string    `gorm:"size:128;not null"`
	APIKeyCipher            string    `gorm:"type:text;column:api_key_cipher"`
	APIKeyLast4             string    `gorm:"size:4;column:api_key_last4"`
	Role                    string    `gorm:"size:256"`
	Description             string    `gorm:"type:text"`
	Active                  bool      `gorm:"default:true"`
	UseForWhatsAppAutoReply bool      `gorm:"default:false;column:use_for_whatsapp_auto_reply"`
	// Resposta em áudio (TTS) na auto-resposta WhatsApp — quando true, não envia texto.
	VoiceReplyEnabled bool   `gorm:"default:false;column:voice_reply_enabled"`
	TTSProvider       string `gorm:"size:32;default:none;column:tts_provider"` // none | openai_tts | gemini_tts | omnivoice | elevenlabs | kokoro
	OpenAITTSVoice    string `gorm:"size:128;column:openai_tts_voice"`         // voz OpenAI / Gemini (Kore…) / voice_id ElevenLabs / voz Kokoro / clone:… OmniVoice
	OpenAITTSModel    string `gorm:"size:64;column:openai_tts_model"`            // modelo TTS (OpenAI ou Gemini conforme tts_provider; vazio = default servidor)
	OpenAITTSAPICipher string `gorm:"type:text;column:openai_tts_api_key_cipher"`
	OpenAITTSAPILast4  string `gorm:"size:4;column:openai_tts_api_key_last4"`
	GeminiTTSAPICipher string `gorm:"type:text;column:gemini_tts_api_key_cipher"`
	GeminiTTSAPILast4  string `gorm:"size:4;column:gemini_tts_api_key_last4"`
	OmnivoiceBaseURL   string `gorm:"size:512;column:omnivoice_base_url"`
	KokoroBaseURL      string `gorm:"size:512;column:kokoro_base_url"`
	ElevenLabsAPICipher string `gorm:"type:text;column:elevenlabs_api_key_cipher"`
	ElevenLabsAPILast4  string `gorm:"size:4;column:elevenlabs_api_key_last4"`
	// Relativo a MEDIA_PERSISTENT_DIR (ex.: agent_voice_previews/<uuid>.wav).
	VoicePreviewPath string `gorm:"size:512;column:voice_preview_path"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (a *AIAgent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
