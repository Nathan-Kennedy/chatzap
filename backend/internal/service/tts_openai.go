package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/model"
)

// Provedores TTS para agentes (campo AIAgent.TTSProvider).
const (
	TTSProviderNone       = "none"
	TTSProviderOpenAI     = "openai_tts"
	TTSProviderOmnivoice  = "omnivoice"
	TTSProviderElevenLabs = "elevenlabs"
	TTSProviderKokoro     = "kokoro"
)

// DefaultOpenAITTSModel modelo OpenAI TTS (POST /v1/audio/speech).
const DefaultOpenAITTSModel = "gpt-4o-mini-tts"

// OmnivoiceClientModel corpo JSON "model" enviado ao omnivoice-server (compat. OpenAI).
const OmnivoiceClientModel = "tts-1"

// MaxOpenAITTSInputRunes limite documentado da API /v1/audio/speech.
const MaxOpenAITTSInputRunes = 4096

type openaiSpeechRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Instructions   *string `json:"instructions,omitempty"`
}

// omnivoiceSpeechRequest — omnivoice-server (Pydantic) só aceita response_format wav|pcm;
// não enviamos o campo para usar o default "wav" e evitar 422 por incompatibilidade com o payload OpenAI (mp3).
type omnivoiceSpeechRequest struct {
	Model   string   `json:"model"`
	Input   string   `json:"input"`
	Voice   string   `json:"voice"`
	Speed   *float64 `json:"speed,omitempty"`
	NumStep *int     `json:"num_step,omitempty"`
}

// omnivoiceDesignFemaleInstructDefault — só a parte após "design:"; tokens em inglês da lista do modelo OmniVoice
// (ex.: erro "Valid English items" no servidor). Vírgula + espaço entre itens.
// "low pitch" tende a soar menos metálico que "moderate pitch" só.
const omnivoiceDesignFemaleInstructDefault = "female, young adult, portuguese accent, low pitch"

// OmnivoiceSpeechOpts parâmetros extra do POST /v1/audio/speech (omnivoice-server).
type OmnivoiceSpeechOpts struct {
	// NaturalFemaleTimbre mapeia vozes nova/shimmer para design:+instruct.
	NaturalFemaleTimbre bool
	// DesignInstructOverride substitui omnivoiceDesignFemaleInstructDefault (ex. vindo de OMNIVOICE_DESIGN_INSTRUCT).
	DesignInstructOverride string
	Speed                  *float64
	NumStep                *int
}

// OmnivoiceAutoReplyDefaults — base para auto-resposta (sem ler .env).
func OmnivoiceAutoReplyDefaults() *OmnivoiceSpeechOpts {
	// Speed < 1 acalma a entonação (menos “agudo”/exaltado no ouvinte).
	sp := 0.90
	ns := 48
	return &OmnivoiceSpeechOpts{
		NaturalFemaleTimbre: true,
		Speed:               &sp,
		NumStep:             &ns,
	}
}

// Nomes de voz só válidos na API OpenAI TTS — não podem ir como voice_id para ElevenLabs/Kokoro.
var openAITTSVoiceNames = map[string]struct{}{
	"alloy": {}, "ash": {}, "ballad": {}, "coral": {}, "echo": {}, "fable": {},
	"marin": {}, "nova": {}, "onyx": {}, "sage": {}, "shimmer": {}, "verse": {}, "cedar": {},
}

func isOpenAITTSVoiceName(s string) bool {
	_, ok := openAITTSVoiceNames[strings.ToLower(strings.TrimSpace(s))]
	return ok
}

// looksLikeElevenLabsVoiceID heurística para IDs públicos (ex. 21m00Tcm4TlvDq8ikWAM).
func looksLikeElevenLabsVoiceID(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) < 18 || len(v) > 32 {
		return false
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		isAZ := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		is09 := c >= '0' && c <= '9'
		if !isAZ && !is09 {
			return false
		}
	}
	return true
}

// SanitizeVoiceForTTSProvider evita reutilizar voz de outro motor (ex. "nova" em ElevenLabs → falha API e sem prévia).
func SanitizeVoiceForTTSProvider(ttsProv string, voice string, cfg *config.Config) string {
	p := NormalizeTTSProvider(ttsProv)
	v := strings.TrimSpace(voice)
	if v == "" {
		return defaultVoiceWhenEmpty(p, cfg)
	}
	lv := strings.ToLower(v)
	switch p {
	case TTSProviderElevenLabs:
		if isOpenAITTSVoiceName(v) || strings.HasPrefix(lv, "clone:") || strings.HasPrefix(lv, "design:") {
			return "21m00Tcm4TlvDq8ikWAM" // Rachel (premade)
		}
		// códigos estilo Kokoro (pf_dora) não são voice_id ElevenLabs
		if strings.Contains(v, "_") && len(v) < 24 {
			return "21m00Tcm4TlvDq8ikWAM"
		}
		return v
	case TTSProviderKokoro:
		if isOpenAITTSVoiceName(v) || strings.HasPrefix(lv, "clone:") || strings.HasPrefix(lv, "design:") {
			return "pf_dora"
		}
		if looksLikeElevenLabsVoiceID(v) {
			return "pf_dora"
		}
		return v
	default:
		return v
	}
}

func defaultVoiceWhenEmpty(p string, cfg *config.Config) string {
	if p == TTSProviderOmnivoice && cfg != nil {
		if dv := strings.TrimSpace(cfg.OmnivoiceDefaultTTSVoice); dv != "" {
			return dv
		}
	}
	if p == TTSProviderElevenLabs {
		return "21m00Tcm4TlvDq8ikWAM"
	}
	if p == TTSProviderKokoro {
		return "pf_dora"
	}
	return "nova"
}

// ResolveStoredOpenAITTSVoice normaliza voz na criação/patch: vazio + OmniVoice usa OMNIVOICE_DEFAULT_TTS_VOICE, senão nova.
func ResolveStoredOpenAITTSVoice(ttsProv string, voiceFromBody string, cfg *config.Config) string {
	v := strings.TrimSpace(voiceFromBody)
	p := NormalizeTTSProvider(ttsProv)
	if v == "" {
		return defaultVoiceWhenEmpty(p, cfg)
	}
	return SanitizeVoiceForTTSProvider(p, v, cfg)
}

// EffectiveOpenAITTSVoice voz efetiva para TTS (campo do agente ou defaults OmniVoice).
func EffectiveOpenAITTSVoice(agent *model.AIAgent, cfg *config.Config) string {
	if agent == nil {
		return ResolveStoredOpenAITTSVoice("", "", cfg)
	}
	return ResolveStoredOpenAITTSVoice(agent.TTSProvider, agent.OpenAITTSVoice, cfg)
}

// EffectiveOpenAITTSModel modelo OpenAI TTS: coluna do agente, depois env, depois default.
func EffectiveOpenAITTSModel(agent *model.AIAgent, cfg *config.Config) string {
	if agent != nil {
		if m := strings.TrimSpace(agent.OpenAITTSModel); m != "" {
			return m
		}
	}
	if cfg != nil {
		if m := strings.TrimSpace(cfg.OpenAITTSModel); m != "" {
			return m
		}
	}
	return DefaultOpenAITTSModel
}

// OpenAITTSInstructionsPtr texto opcional para o campo instructions da API (tom/estilo).
func OpenAITTSInstructionsPtr(cfg *config.Config) *string {
	if cfg == nil {
		return nil
	}
	s := strings.TrimSpace(cfg.OpenAITTSInstructions)
	if s == "" {
		return nil
	}
	return &s
}

// EffectiveKokoroTTSModel corpo "model" para servidores Kokoro compatíveis (ex. Kokoro-FastAPI).
func EffectiveKokoroTTSModel(cfg *config.Config) string {
	if cfg != nil {
		if m := strings.TrimSpace(cfg.KokoroTTSModel); m != "" {
			return m
		}
	}
	return "kokoro"
}

// EffectiveElevenLabsModel modelo ElevenLabs (ex. eleven_multilingual_v2).
func EffectiveElevenLabsModel(cfg *config.Config) string {
	if cfg != nil {
		if m := strings.TrimSpace(cfg.ElevenLabsDefaultModel); m != "" {
			return m
		}
	}
	return "eleven_multilingual_v2"
}

// OmnivoiceAutoReplyOptsFromConfig aplica OMNIVOICE_DESIGN_INSTRUCT, OMNIVOICE_TTS_SPEED, OMNIVOICE_TTS_NUM_STEP.
func OmnivoiceAutoReplyOptsFromConfig(cfg *config.Config) *OmnivoiceSpeechOpts {
	o := OmnivoiceAutoReplyDefaults()
	if cfg == nil {
		return o
	}
	if s := strings.TrimSpace(cfg.OmnivoiceDesignInstruct); s != "" {
		o.DesignInstructOverride = s
	}
	if cfg.OmnivoiceTTSNumStep > 0 {
		n := cfg.OmnivoiceTTSNumStep
		o.NumStep = &n
	}
	if cfg.OmnivoiceTTSSpeed > 0 {
		sp := cfg.OmnivoiceTTSSpeed
		o.Speed = &sp
	}
	return o
}

// mapVoiceForOmnivoice mapeia vozes preset da OpenAI TTS para "auto" (OmniVoice interpreta nomes desconhecidos como modo design).
func mapVoiceForOmnivoice(v string) string {
	return mapVoiceForOmnivoiceWithOpts(v, nil)
}

func mapVoiceForOmnivoiceWithOpts(v string, opts *OmnivoiceSpeechOpts) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return "auto"
	}
	low := strings.ToLower(s)
	if opts != nil && opts.NaturalFemaleTimbre && (low == "nova" || low == "shimmer") {
		instruct := omnivoiceDesignFemaleInstructDefault
		if t := strings.TrimSpace(opts.DesignInstructOverride); t != "" {
			instruct = t
		}
		return "design:" + instruct
	}
	switch low {
	case "nova", "shimmer", "alloy", "echo", "fable", "onyx":
		return "auto"
	default:
		return s
	}
}

// SynthOpenAITTS gera MP3 (audio/mpeg) via API OpenAI.
func SynthOpenAITTS(ctx context.Context, apiKey, model, voice, text string, instructions *string) ([]byte, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("openai tts: api key em falta")
	}
	if model == "" {
		model = DefaultOpenAITTSModel
	}
	voice = strings.TrimSpace(voice)
	if voice == "" {
		voice = "nova"
	}
	body, err := json.Marshal(openaiSpeechRequest{
		Model:          model,
		Input:          text,
		Voice:          voice,
		ResponseFormat: "mp3",
		Instructions:   instructions,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/audio/speech", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	hc := &http.Client{Timeout: 120 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 25<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai tts status %d: %s", resp.StatusCode, truncateErrBody(raw))
	}
	return raw, nil
}

// SynthOmnivoiceOpenAICompat chama POST {baseURL}/v1/audio/speech (compat. OpenAI / OmniVoice). opts pode ser nil.
func SynthOmnivoiceOpenAICompat(ctx context.Context, baseURL, bearerToken, model, voice, text string, opts *OmnivoiceSpeechOpts) ([]byte, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("omnivoice: base URL em falta")
	}
	if model == "" {
		model = OmnivoiceClientModel
	}
	voiceMapped := mapVoiceForOmnivoiceWithOpts(strings.TrimSpace(voice), opts)
	reqBody := omnivoiceSpeechRequest{
		Model: model,
		Input: text,
		Voice: voiceMapped,
	}
	if opts != nil {
		reqBody.Speed = opts.Speed
		reqBody.NumStep = opts.NumStep
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	u := base + "/v1/audio/speech"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if t := strings.TrimSpace(bearerToken); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	hc := &http.Client{Timeout: 180 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 30<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("omnivoice http %d: %s", resp.StatusCode, truncateErrBody(raw))
	}
	return raw, nil
}

// kokoroSpeechRequest corpo mínimo para servidores Kokoro compatíveis com OpenAI (ex. Kokoro-FastAPI).
type kokoroSpeechRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Voice          string `json:"voice"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// SynthKokoroOpenAICompat chama POST {baseURL}/v1/audio/speech sem mapeamento de voz (voice = id Kokoro, ex. pf_dora).
func SynthKokoroOpenAICompat(ctx context.Context, baseURL, bearerToken, model, voice, text string) ([]byte, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("kokoro: base URL em falta")
	}
	if model == "" {
		model = "kokoro"
	}
	voice = strings.TrimSpace(voice)
	if voice == "" {
		return nil, fmt.Errorf("kokoro: voice em falta")
	}
	reqBody := kokoroSpeechRequest{
		Model: model,
		Input: text,
		Voice: voice,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	u := base + "/v1/audio/speech"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if t := strings.TrimSpace(bearerToken); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	hc := &http.Client{Timeout: 180 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 30<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("kokoro http %d: %s", resp.StatusCode, truncateErrBody(raw))
	}
	return raw, nil
}

func truncateErrBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
}

// TruncateForTTS corta o texto ao limite de runas da API TTS.
func TruncateForTTS(text string, maxRunes int) string {
	if maxRunes < 1 {
		return ""
	}
	r := []rune(text)
	if len(r) <= maxRunes {
		return text
	}
	return string(r[:maxRunes])
}
