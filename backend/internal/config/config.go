package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config carrega variáveis de ambiente (12-factor). Nunca commitar segredos.
type Config struct {
	Env              string
	HTTPPort         string
	DatabaseURL      string
	RedisURL         string
	LogLevel         string
	CORSAllowOrigins []string

	// WhatsApp: evolution (Evolution API), baileys (gateway Node), none (só webhooks / testes sem envio REST)
	WhatsAppProvider string

	// Evolution Go (envio + validação de webhook) — obrigatório se WhatsAppProvider=evolution
	EvolutionBaseURL      string
	EvolutionAPIKey       string // chave global admin; também fallback de envio
	EvolutionInstanceName string // fallback para token de instância em rotas internas legadas

	// URL base onde a API expõe /webhooks/whatsapp/:instance (Evolution no Docker: host.docker.internal)
	PublicWebhookBaseURL string

	// URL base onde a Evolution faz GET do ficheiro (ex. http://host.docker.internal:8088). Vazio = igual a PublicWebhookBaseURL.
	PublicMediaBaseURL string
	// Diretório para uploads temporários antes do /send/media
	MediaUploadDir string
	// Tamanho máximo de upload (bytes)
	MediaMaxUploadBytes int64
	// TTL do token público /media/temp/:token
	MediaTokenTTLMinutes int
	// Cópia persistente por message_id (miniaturas / áudio / vídeo na Inbox)
	MediaPersistentDir string

	// Webhook: use pelo menos um mecanismo em ambientes não-dev.
	WebhookSharedSecret     string // header X-Webhook-Secret (recomendado)
	EvolutionWebhookAPIKey  string // compara com campo "apikey" no JSON (Evolution envia)
	InsecureSkipWebhookAuth bool   // só local; exige ENV=development

	// Rota interna de teste (substituir por JWT em produção)
	InternalAPIKey string

	// LLM: gemini (padrão) ou openai — ver LLM_PROVIDER
	LLMProvider      string
	GeminiAPIKey     string
	GeminiModel      string
	// Gemini TTS (Speech API): modelo e instrução antes do texto (opcional; vazio = preset no código).
	GeminiTTSModel       string
	GeminiTTSInstruction string
	OpenAIAPIKey     string
	OpenAIModel      string
	LLMSystemPrompt  string // partilhado entre Gemini e OpenAI
	AutoReplyEnabled bool

	// Opcional: restringir :instance_id na URL a estes valores (vazio = qualquer)
	AllowedInstanceIDs map[string]struct{}

	// JWT (auth utilizador)
	JWTSecret           string
	JWTAccessTTLMinutes int
	JWTRefreshTTLDays   int

	// Encriptação de segredos por workspace (ex. API keys de agentes IA). Opcional até criar o primeiro agente.
	AppEncryptionKey string

	// URL base do servidor OmniVoice (compat. POST /v1/audio/speech) quando o agente não define omnivoice_base_url.
	OmnivoiceDefaultBaseURL string
	// Voz OmniVoice quando openai_tts_voice do agente está vazio (ex.: clone:atendimento_br para sotaque BR via perfil no servidor).
	OmnivoiceDefaultTTSVoice string
	// Instrução modo design (só tokens em inglês permitidos pelo modelo; vírgula + espaço). Vazio = preset no código.
	OmnivoiceDesignInstruct string
	// 0 = usar defaults da auto-resposta (speed ~0.90, num_step ~48).
	OmnivoiceTTSSpeed   float64
	OmnivoiceTTSNumStep int

	// OpenAI TTS (POST /v1/audio/speech)
	OpenAITTSModel        string // vazio = gpt-4o-mini-tts
	OpenAITTSInstructions string // opcional → campo instructions na API
	// ElevenLabs TTS
	ElevenLabsDefaultModel string // ex. eleven_multilingual_v2
	// ElevenLabsAPIKey xi-api-key global (fallback se o agente não tiver chave na UI). Não logar.
	ElevenLabsAPIKey string
	// ElevenLabsAPIBaseURL raiz da API (residência EU/US/in: ver documentação). Não logar.
	ElevenLabsAPIBaseURL string
	// ConvAI + Twilio: outbound call (https://elevenlabs.io/docs/api-reference/twilio/outbound-call)
	ElevenLabsConvAIAgentID            string
	ElevenLabsConvAIAgentPhoneNumberID string
	// Kokoro (servidor compat. OpenAI, ex. Kokoro-FastAPI)
	KokoroDefaultBaseURL string
	KokoroTTSModel       string // corpo JSON "model" (ex. kokoro)
	// Pausa entre fim do envio de áudio TTS e mensagem de texto de resumo (ms). 0 = sem pausa extra.
	VoiceToTextGapMs int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	env := get("ENV", "development")
	httpPort := get("HTTP_PORT", "8080")
	publicWh := strings.TrimSpace(os.Getenv("PUBLIC_WEBHOOK_BASE_URL"))
	if publicWh == "" {
		// Evolution em contentor precisa de alcançar a API no host (Windows/Mac Docker Desktop).
		publicWh = "http://host.docker.internal:" + httpPort
	}
	mediaBase := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_MEDIA_BASE_URL")), "/")
	if mediaBase == "" {
		mediaBase = strings.TrimRight(publicWh, "/")
	}
	mediaDir := strings.TrimSpace(os.Getenv("MEDIA_UPLOAD_DIR"))
	if mediaDir == "" {
		mediaDir = ".tmp/media_uploads"
	}
	maxMedia := int64(25 * 1024 * 1024)
	if v := strings.TrimSpace(os.Getenv("MEDIA_MAX_UPLOAD_BYTES")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxMedia = n
		}
	}
	mediaTTL := parseInt(get("MEDIA_TOKEN_TTL_MINUTES", "30"), 30)
	persistDir := strings.TrimSpace(os.Getenv("MEDIA_PERSISTENT_DIR"))
	if persistDir == "" {
		persistDir = ".tmp/message_media"
	}

	c := &Config{
		Env:                                env,
		HTTPPort:                           httpPort,
		PublicWebhookBaseURL:               strings.TrimRight(publicWh, "/"),
		PublicMediaBaseURL:                 mediaBase,
		MediaUploadDir:                     mediaDir,
		MediaMaxUploadBytes:                maxMedia,
		MediaTokenTTLMinutes:               mediaTTL,
		MediaPersistentDir:                 persistDir,
		DatabaseURL:                        os.Getenv("DATABASE_URL"),
		RedisURL:                           get("REDIS_URL", "redis://127.0.0.1:6379/0"),
		LogLevel:                           get("LOG_LEVEL", "info"),
		CORSAllowOrigins:                   splitCSV(get("CORS_ALLOW_ORIGINS", "http://localhost:5173")),
		WhatsAppProvider:                   normalizeWhatsAppProvider(get("WHATSAPP_PROVIDER", "evolution")),
		EvolutionBaseURL:                   strings.TrimRight(os.Getenv("EVOLUTION_BASE_URL"), "/"),
		EvolutionAPIKey:                    os.Getenv("EVOLUTION_API_KEY"),
		EvolutionInstanceName:              get("EVOLUTION_INSTANCE_NAME", "default"),
		WebhookSharedSecret:                os.Getenv("WEBHOOK_SHARED_SECRET"),
		EvolutionWebhookAPIKey:             os.Getenv("EVOLUTION_WEBHOOK_API_KEY"),
		InsecureSkipWebhookAuth:            parseBool(get("INSECURE_SKIP_WEBHOOK_AUTH", "false")),
		InternalAPIKey:                     os.Getenv("INTERNAL_API_KEY"),
		LLMProvider:                        normalizeLLMProvider(get("LLM_PROVIDER", "gemini")),
		GeminiAPIKey:                       os.Getenv("GEMINI_API_KEY"),
		GeminiModel:                        get("GEMINI_MODEL", "gemini-2.5-flash"),
		GeminiTTSModel:                     strings.TrimSpace(os.Getenv("GEMINI_TTS_MODEL")),
		GeminiTTSInstruction:               strings.TrimSpace(os.Getenv("GEMINI_TTS_INSTRUCTION")),
		OpenAIAPIKey:                       os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:                        get("OPENAI_MODEL", "gpt-4o-mini"),
		LLMSystemPrompt:                    getLLMSystemPrompt(),
		AutoReplyEnabled:                   parseBool(get("AUTO_REPLY_ENABLED", "true")),
		JWTSecret:                          os.Getenv("JWT_SECRET"),
		JWTAccessTTLMinutes:                parseInt(get("JWT_ACCESS_TTL_MINUTES", "1440"), 1440),
		JWTRefreshTTLDays:                  parseInt(get("JWT_REFRESH_TTL_DAYS", "7"), 7),
		AppEncryptionKey:                   strings.TrimSpace(os.Getenv("APP_ENCRYPTION_KEY")),
		OmnivoiceDefaultBaseURL:            strings.TrimRight(strings.TrimSpace(os.Getenv("OMNIVOICE_DEFAULT_BASE_URL")), "/"),
		OmnivoiceDefaultTTSVoice:           strings.TrimSpace(os.Getenv("OMNIVOICE_DEFAULT_TTS_VOICE")),
		OmnivoiceDesignInstruct:            strings.TrimSpace(os.Getenv("OMNIVOICE_DESIGN_INSTRUCT")),
		OmnivoiceTTSSpeed:                  parseFloatNonNeg(os.Getenv("OMNIVOICE_TTS_SPEED")),
		OmnivoiceTTSNumStep:                parseIntNonNeg(os.Getenv("OMNIVOICE_TTS_NUM_STEP")),
		OpenAITTSModel:                     strings.TrimSpace(os.Getenv("OPENAI_TTS_MODEL")),
		OpenAITTSInstructions:              strings.TrimSpace(os.Getenv("OPENAI_TTS_INSTRUCTIONS")),
		ElevenLabsDefaultModel:             strings.TrimSpace(os.Getenv("ELEVENLABS_DEFAULT_MODEL")),
		ElevenLabsAPIKey:                   strings.TrimSpace(os.Getenv("ELEVENLABS_API_KEY")),
		ElevenLabsAPIBaseURL:               strings.TrimRight(strings.TrimSpace(os.Getenv("ELEVENLABS_API_BASE_URL")), "/"),
		ElevenLabsConvAIAgentID:            strings.TrimSpace(os.Getenv("ELEVENLABS_CONVAI_AGENT_ID")),
		ElevenLabsConvAIAgentPhoneNumberID: strings.TrimSpace(os.Getenv("ELEVENLABS_CONVAI_AGENT_PHONE_NUMBER_ID")),
		KokoroDefaultBaseURL:               strings.TrimRight(strings.TrimSpace(os.Getenv("KOKORO_DEFAULT_BASE_URL")), "/"),
		KokoroTTSModel:                     strings.TrimSpace(os.Getenv("KOKORO_TTS_MODEL")),
		VoiceToTextGapMs:                   parseInt(get("VOICE_TO_TEXT_GAP_MS", "1200"), 1200),
	}

	if raw := os.Getenv("ALLOWED_INSTANCE_IDS"); raw != "" {
		c.AllowedInstanceIDs = make(map[string]struct{})
		for _, id := range splitCSV(raw) {
			id = strings.TrimSpace(id)
			if id != "" {
				c.AllowedInstanceIDs[id] = struct{}{}
			}
		}
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL é obrigatório")
	}
	if c.WhatsAppProvider == "" {
		return nil, fmt.Errorf("WHATSAPP_PROVIDER inválido (use evolution, baileys ou none)")
	}
	if c.WhatsAppProvider == "evolution" {
		if c.EvolutionBaseURL == "" {
			return nil, fmt.Errorf("EVOLUTION_BASE_URL é obrigatório quando WHATSAPP_PROVIDER=evolution")
		}
		if c.EvolutionAPIKey == "" {
			return nil, fmt.Errorf("EVOLUTION_API_KEY é obrigatório quando WHATSAPP_PROVIDER=evolution")
		}
	}
	if c.InternalAPIKey == "" {
		return nil, fmt.Errorf("INTERNAL_API_KEY é obrigatório (rota de envio de teste)")
	}
	if strings.TrimSpace(c.JWTSecret) == "" {
		return nil, fmt.Errorf("JWT_SECRET é obrigatório (auth de utilizadores)")
	}
	if len(c.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET deve ter pelo menos 32 caracteres")
	}
	// AUTO_REPLY_ENABLED liga o fluxo de auto-resposta no webhook; a chave pode vir só de agentes por
	// workspace (BD). Chaves em .env são opcionais e servem de fallback quando não há agente.

	if c.WebhookSharedSecret == "" && c.EvolutionWebhookAPIKey == "" {
		if c.InsecureSkipWebhookAuth {
			if env != "development" {
				return nil, fmt.Errorf("INSECURE_SKIP_WEBHOOK_AUTH só é permitido com ENV=development")
			}
		} else {
			return nil, fmt.Errorf("defina WEBHOOK_SHARED_SECRET e/ou EVOLUTION_WEBHOOK_API_KEY, ou INSECURE_SKIP_WEBHOOK_AUTH=true apenas em dev local")
		}
	}

	return c, nil
}

func get(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseBool(s string) bool {
	b, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(s)))
	return err == nil && b
}

func parseInt(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// parseIntNonNeg devolve 0 se vazio ou inválido (para overrides opcionais).
func parseIntNonNeg(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// parseFloatNonNeg devolve 0 se vazio ou inválido.
func parseFloatNonNeg(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 {
		return 0
	}
	return f
}

func normalizeLLMProvider(s string) string {
	p := strings.ToLower(strings.TrimSpace(s))
	if p == "" {
		return "gemini"
	}
	return p
}

// normalizeWhatsAppProvider: evolution (padrão), baileys, none.
func normalizeWhatsAppProvider(s string) string {
	p := strings.ToLower(strings.TrimSpace(s))
	if p == "" {
		return "evolution"
	}
	switch p {
	case "evolution", "baileys", "none":
		return p
	default:
		return ""
	}
}

// getLLMSystemPrompt: LLM_SYSTEM_PROMPT tem prioridade; mantém compat com OPENAI_SYSTEM_PROMPT.
func getLLMSystemPrompt() string {
	if v := strings.TrimSpace(os.Getenv("LLM_SYSTEM_PROMPT")); v != "" {
		return v
	}
	return get("OPENAI_SYSTEM_PROMPT", "Você é um assistente de atendimento por WhatsApp no Brasil. Responda em português brasileiro (pt-BR), de forma curta, clara e cordial. Não use Markdown nem asteriscos (* ou **) para negrito ou listas; o WhatsApp não formata isso — escreva texto simples. Se receber histórico da conversa no mesmo pedido, use-o como contexto. Use o nome do cliente só quando for natural (saudação, proposta, ou se ele perguntar se você lembra dele); não repita o nome o tempo todo. Não invente diminutivos do nome (ex.: «Nathanzinho») se o cliente não usou esse tratamento. Se já cumprimentou antes na conversa, não comece de novo com «Olá» nem com o nome — siga o assunto. Use emojis com moderação: prefira nenhum ou no máximo um por resposta; tom profissional vem antes de decoração.")
}

// WebhookURLForWhatsAppInstance URL completa que o Evolution Go deve chamar (path com nome técnico da instância).
func (c *Config) WebhookURLForWhatsAppInstance(instanceSlug string) string {
	s := strings.TrimSpace(instanceSlug)
	if s == "" || c.PublicWebhookBaseURL == "" {
		return ""
	}
	return c.PublicWebhookBaseURL + "/webhooks/whatsapp/" + url.PathEscape(s)
}

// MediaTempFetchURL URL pública GET /media/temp/:token (acessível pelo contentor Evolution).
func (c *Config) MediaTempFetchURL(token string) string {
	t := strings.TrimSpace(token)
	if t == "" || c.PublicMediaBaseURL == "" {
		return ""
	}
	return c.PublicMediaBaseURL + "/media/temp/" + url.PathEscape(t)
}
