package service

import (
	"fmt"
	"strings"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/cryptoagent"
	"wa-saas/backend/internal/model"
)

// ResolveOpenAITTSAPIKey devolve a chave OpenAI para TTS: campo dedicado ou a mesma do agente se provider openai.
func ResolveOpenAITTSAPIKey(encKey string, a *model.AIAgent) (string, error) {
	if a == nil {
		return "", fmt.Errorf("agente nil")
	}
	if strings.TrimSpace(a.OpenAITTSAPICipher) != "" {
		return cryptoagent.Decrypt(a.OpenAITTSAPICipher, encKey)
	}
	if strings.ToLower(strings.TrimSpace(a.Provider)) == "openai" && strings.TrimSpace(a.APIKeyCipher) != "" {
		return cryptoagent.Decrypt(a.APIKeyCipher, encKey)
	}
	return "", fmt.Errorf("sem chave OpenAI para TTS (defina openai_tts_api_key ou use agente com provider openai)")
}

// ResolveOmnivoiceBaseURL usa a URL do agente; se vazia, o default do servidor (OMNIVOICE_DEFAULT_BASE_URL).
func ResolveOmnivoiceBaseURL(a *model.AIAgent, defaultBaseURL string) string {
	if a != nil {
		if b := strings.TrimSpace(a.OmnivoiceBaseURL); b != "" {
			return strings.TrimRight(b, "/")
		}
	}
	return strings.TrimRight(strings.TrimSpace(defaultBaseURL), "/")
}

// ResolveKokoroBaseURL URL do Kokoro-FastAPI ou compat.; fallback KOKORO_DEFAULT_BASE_URL.
func ResolveKokoroBaseURL(a *model.AIAgent, defaultBaseURL string) string {
	if a != nil {
		if b := strings.TrimSpace(a.KokoroBaseURL); b != "" {
			return strings.TrimRight(b, "/")
		}
	}
	return strings.TrimRight(strings.TrimSpace(defaultBaseURL), "/")
}

// ResolveElevenLabsAPIKey devolve xi-api-key: do agente (encriptada) ou ELEVENLABS_API_KEY no servidor.
func ResolveElevenLabsAPIKey(encKey string, cfg *config.Config, a *model.AIAgent) (string, error) {
	if a != nil && strings.TrimSpace(a.ElevenLabsAPICipher) != "" {
		return cryptoagent.Decrypt(a.ElevenLabsAPICipher, encKey)
	}
	if cfg != nil {
		if k := strings.TrimSpace(cfg.ElevenLabsAPIKey); k != "" {
			return k, nil
		}
	}
	return "", fmt.Errorf("sem chave ElevenLabs (defina no agente ou ELEVENLABS_API_KEY no servidor)")
}

// NormalizeTTSProvider devolve none se vazio.
func NormalizeTTSProvider(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "", TTSProviderNone:
		return TTSProviderNone
	case TTSProviderOpenAI, TTSProviderOmnivoice, TTSProviderElevenLabs, TTSProviderKokoro:
		return s
	default:
		return TTSProviderNone
	}
}
