package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/cryptoagent"
	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
	"wa-saas/backend/internal/service"
)

func agentToDTO(a model.AIAgent, cfg *config.Config) fiber.Map {
	last4 := a.APIKeyLast4
	if last4 == "" && a.APIKeyCipher != "" {
		last4 = "****"
	}
	ttsLast4 := a.OpenAITTSAPILast4
	if ttsLast4 == "" && a.OpenAITTSAPICipher != "" {
		ttsLast4 = "****"
	}
	elLast4 := a.ElevenLabsAPILast4
	if elLast4 == "" && a.ElevenLabsAPICipher != "" {
		elLast4 = "****"
	}
	hasElAgent := strings.TrimSpace(a.ElevenLabsAPICipher) != ""
	hasElEnv := cfg != nil && strings.TrimSpace(cfg.ElevenLabsAPIKey) != ""
	if !hasElAgent && hasElEnv && elLast4 == "" {
		elLast4 = last4FromKey(cfg.ElevenLabsAPIKey)
	}
	return fiber.Map{
		"id":                          a.ID.String(),
		"name":                        a.Name,
		"provider":                    a.Provider,
		"model":                       a.Model,
		"has_api_key":                 strings.TrimSpace(a.APIKeyCipher) != "",
		"api_key_last4":               last4,
		"role":                        a.Role,
		"description":                 a.Description,
		"active":                      a.Active,
		"use_for_whatsapp_auto_reply": a.UseForWhatsAppAutoReply,
		"voice_reply_enabled":         a.VoiceReplyEnabled,
		"tts_provider":                service.NormalizeTTSProvider(a.TTSProvider),
		"openai_tts_voice":            strings.TrimSpace(a.OpenAITTSVoice),
		"openai_tts_model":            strings.TrimSpace(a.OpenAITTSModel),
		"has_openai_tts_api_key":      strings.TrimSpace(a.OpenAITTSAPICipher) != "",
		"openai_tts_api_key_last4":    ttsLast4,
		"omnivoice_base_url":          strings.TrimSpace(a.OmnivoiceBaseURL),
		"kokoro_base_url":             strings.TrimSpace(a.KokoroBaseURL),
		"has_elevenlabs_api_key":      hasElAgent || hasElEnv,
		"elevenlabs_api_key_last4":    elLast4,
		"voice_preview_available":     strings.TrimSpace(a.VoicePreviewPath) != "",
		"created_at":                  a.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updated_at":                  a.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func last4FromKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	runes := []rune(key)
	if len(runes) <= 4 {
		return string(runes)
	}
	return string(runes[len(runes)-4:])
}

// HandleListAgents GET /api/v1/agents
func HandleListAgents(db *gorm.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		var list []model.AIAgent
		if err := db.Where("workspace_id = ?", wid).Order("created_at DESC").Find(&list).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		out := make([]fiber.Map, 0, len(list))
		for _, a := range list {
			out = append(out, agentToDTO(a, cfg))
		}
		return JSONSuccess(c, out)
	}
}

type createAgentBody struct {
	Name                    string `json:"name"`
	Provider                string `json:"provider"`
	Model                   string `json:"model"`
	APIKey                  string `json:"api_key"`
	Role                    string `json:"role"`
	Description             string `json:"description"`
	Active                  *bool  `json:"active"`
	UseForWhatsAppAutoReply bool   `json:"use_for_whatsapp_auto_reply"`
	VoiceReplyEnabled       bool   `json:"voice_reply_enabled"`
	TTSProvider             string `json:"tts_provider"`
	OpenAITTSVoice          string `json:"openai_tts_voice"`
	OpenAITTSModel          string `json:"openai_tts_model"`
	OpenAITTSAPIKey         string `json:"openai_tts_api_key"`
	OmnivoiceBaseURL        string `json:"omnivoice_base_url"`
	KokoroBaseURL           string `json:"kokoro_base_url"`
	ElevenLabsAPIKey        string `json:"elevenlabs_api_key"`
}

// HandleCreateAgent POST /api/v1/agents
func HandleCreateAgent(log *zap.Logger, db *gorm.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		if strings.TrimSpace(cfg.AppEncryptionKey) == "" {
			return JSONError(c, fiber.StatusServiceUnavailable, "encryption_not_configured",
				"defina APP_ENCRYPTION_KEY no servidor para guardar agentes com chave API encriptada", nil)
		}
		var body createAgentBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "", nil)
		}
		name := strings.TrimSpace(body.Name)
		provider := strings.ToLower(strings.TrimSpace(body.Provider))
		modelName := strings.TrimSpace(body.Model)
		apiKey := strings.TrimSpace(body.APIKey)
		if name == "" || provider == "" || modelName == "" || apiKey == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "name, provider, model e api_key são obrigatórios", nil)
		}
		if provider != "gemini" && provider != "openai" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "provider deve ser gemini ou openai", nil)
		}
		ttsProv := service.NormalizeTTSProvider(body.TTSProvider)
		if !body.VoiceReplyEnabled {
			ttsProv = service.TTSProviderNone
		}
		if body.VoiceReplyEnabled && ttsProv == service.TTSProviderNone {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "escolha um provedor TTS ou desative resposta em voz", nil)
		}
		if ttsProv == service.TTSProviderOpenAI {
			ttsKey := strings.TrimSpace(body.OpenAITTSAPIKey)
			if provider == "gemini" && ttsKey == "" {
				return JSONError(c, fiber.StatusBadRequest, "validation_error",
					"openai_tts_api_key obrigatório para TTS OpenAI quando o LLM é Gemini", nil)
			}
		}
		if ttsProv == service.TTSProviderOmnivoice {
			omni := strings.TrimSpace(body.OmnivoiceBaseURL)
			if omni == "" {
				omni = strings.TrimSpace(cfg.OmnivoiceDefaultBaseURL)
			}
			if omni == "" {
				return JSONError(c, fiber.StatusBadRequest, "validation_error",
					"omnivoice_base_url obrigatório para OmniVoice (ou defina OMNIVOICE_DEFAULT_BASE_URL no servidor)", nil)
			}
		}
		if ttsProv == service.TTSProviderElevenLabs {
			if strings.TrimSpace(body.ElevenLabsAPIKey) == "" && (cfg == nil || strings.TrimSpace(cfg.ElevenLabsAPIKey) == "") {
				return JSONError(c, fiber.StatusBadRequest, "validation_error",
					"elevenlabs_api_key obrigatório para ElevenLabs (ou defina ELEVENLABS_API_KEY no servidor)", nil)
			}
		}
		if ttsProv == service.TTSProviderKokoro {
			kb := strings.TrimSpace(body.KokoroBaseURL)
			if kb == "" {
				kb = strings.TrimSpace(cfg.KokoroDefaultBaseURL)
			}
			if kb == "" {
				return JSONError(c, fiber.StatusBadRequest, "validation_error",
					"kokoro_base_url obrigatório para Kokoro (ou defina KOKORO_DEFAULT_BASE_URL no servidor)", nil)
			}
		}
		cipher, err := cryptoagent.Encrypt(apiKey, cfg.AppEncryptionKey)
		if err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "encrypt_error", err.Error(), nil)
		}
		active := true
		if body.Active != nil {
			active = *body.Active
		}
		voice := service.ResolveStoredOpenAITTSVoice(ttsProv, body.OpenAITTSVoice, cfg)
		agent := model.AIAgent{
			WorkspaceID:             wid,
			Name:                    name,
			Provider:                provider,
			Model:                   modelName,
			APIKeyCipher:            cipher,
			APIKeyLast4:             last4FromKey(apiKey),
			Role:                    strings.TrimSpace(body.Role),
			Description:             strings.TrimSpace(body.Description),
			Active:                  active,
			UseForWhatsAppAutoReply: body.UseForWhatsAppAutoReply,
			VoiceReplyEnabled:       body.VoiceReplyEnabled && ttsProv != service.TTSProviderNone,
			TTSProvider:             ttsProv,
			OpenAITTSVoice:          voice,
			OpenAITTSModel:          strings.TrimSpace(body.OpenAITTSModel),
			OmnivoiceBaseURL:        strings.TrimSpace(body.OmnivoiceBaseURL),
			KokoroBaseURL:           strings.TrimSpace(body.KokoroBaseURL),
		}
		ttsKey := strings.TrimSpace(body.OpenAITTSAPIKey)
		if ttsKey != "" {
			tc, err := cryptoagent.Encrypt(ttsKey, cfg.AppEncryptionKey)
			if err != nil {
				return JSONError(c, fiber.StatusInternalServerError, "encrypt_error", err.Error(), nil)
			}
			agent.OpenAITTSAPICipher = tc
			agent.OpenAITTSAPILast4 = last4FromKey(ttsKey)
		}
		elKey := strings.TrimSpace(body.ElevenLabsAPIKey)
		if elKey != "" {
			ec, err := cryptoagent.Encrypt(elKey, cfg.AppEncryptionKey)
			if err != nil {
				return JSONError(c, fiber.StatusInternalServerError, "encrypt_error", err.Error(), nil)
			}
			agent.ElevenLabsAPICipher = ec
			agent.ElevenLabsAPILast4 = last4FromKey(elKey)
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if agent.UseForWhatsAppAutoReply {
				if err := service.ClearOtherWhatsAppAutoReplyAgents(tx, wid, uuid.Nil); err != nil {
					return err
				}
			}
			return tx.Create(&agent).Error
		}); err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		ctxPrev, cancelPrev := context.WithTimeout(c.Context(), 2*time.Minute)
		defer cancelPrev()
		if err := service.RegenerateAgentVoicePreview(ctxPrev, log, db, cfg, cfg.AppEncryptionKey, &agent); err != nil && log != nil {
			log.Warn("prévia voz (create)", zap.Error(err))
		}
		if err := db.Where("id = ?", agent.ID).First(&agent).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		return JSONSuccess(c, agentToDTO(agent, cfg))
	}
}

type patchAgentBody struct {
	Name                    *string `json:"name"`
	Provider                *string `json:"provider"`
	Model                   *string `json:"model"`
	APIKey                  *string `json:"api_key"`
	Role                    *string `json:"role"`
	Description             *string `json:"description"`
	Active                  *bool   `json:"active"`
	UseForWhatsAppAutoReply *bool   `json:"use_for_whatsapp_auto_reply"`
	VoiceReplyEnabled       *bool   `json:"voice_reply_enabled"`
	TTSProvider             *string `json:"tts_provider"`
	OpenAITTSVoice          *string `json:"openai_tts_voice"`
	OpenAITTSAPIKey         *string `json:"openai_tts_api_key"`
	OmnivoiceBaseURL        *string `json:"omnivoice_base_url"`
	OpenAITTSModel          *string `json:"openai_tts_model"`
	KokoroBaseURL           *string `json:"kokoro_base_url"`
	ElevenLabsAPIKey        *string `json:"elevenlabs_api_key"`
}

// validateAgentTTSAfterPatch valida combinação voz + provedor + chaves após PATCH (estado atual + body).
func validateAgentTTSAfterPatch(cur model.AIAgent, body patchAgentBody, cfg *config.Config) error {
	voice := cur.VoiceReplyEnabled
	if body.VoiceReplyEnabled != nil {
		voice = *body.VoiceReplyEnabled
	}
	tp := service.NormalizeTTSProvider(cur.TTSProvider)
	if body.TTSProvider != nil {
		tp = service.NormalizeTTSProvider(*body.TTSProvider)
	}
	if !voice {
		return nil
	}
	if tp == service.TTSProviderNone {
		return fmt.Errorf("escolha um provedor TTS ou desative resposta em voz")
	}
	prov := strings.ToLower(strings.TrimSpace(cur.Provider))
	if body.Provider != nil {
		prov = strings.ToLower(strings.TrimSpace(*body.Provider))
	}
	hasTTSKey := strings.TrimSpace(cur.OpenAITTSAPICipher) != ""
	if body.OpenAITTSAPIKey != nil && strings.TrimSpace(*body.OpenAITTSAPIKey) != "" {
		hasTTSKey = true
	}
	if tp == service.TTSProviderOpenAI && prov == "gemini" && !hasTTSKey {
		return fmt.Errorf("openai_tts_api_key obrigatório para TTS OpenAI quando o LLM é Gemini")
	}
	hasElKey := strings.TrimSpace(cur.ElevenLabsAPICipher) != ""
	if body.ElevenLabsAPIKey != nil && strings.TrimSpace(*body.ElevenLabsAPIKey) != "" {
		hasElKey = true
	}
	envEl := cfg != nil && strings.TrimSpace(cfg.ElevenLabsAPIKey) != ""
	if tp == service.TTSProviderElevenLabs && !hasElKey && !envEl {
		return fmt.Errorf("elevenlabs_api_key obrigatório para ElevenLabs (ou defina ELEVENLABS_API_KEY no servidor)")
	}
	omniURL := strings.TrimSpace(cur.OmnivoiceBaseURL)
	if body.OmnivoiceBaseURL != nil {
		omniURL = strings.TrimSpace(*body.OmnivoiceBaseURL)
	}
	if omniURL == "" && cfg != nil {
		omniURL = strings.TrimSpace(cfg.OmnivoiceDefaultBaseURL)
	}
	if tp == service.TTSProviderOmnivoice && omniURL == "" {
		return fmt.Errorf("omnivoice_base_url obrigatório para OmniVoice (ou defina OMNIVOICE_DEFAULT_BASE_URL no servidor)")
	}
	kokoroURL := strings.TrimSpace(cur.KokoroBaseURL)
	if body.KokoroBaseURL != nil {
		kokoroURL = strings.TrimSpace(*body.KokoroBaseURL)
	}
	if kokoroURL == "" && cfg != nil {
		kokoroURL = strings.TrimSpace(cfg.KokoroDefaultBaseURL)
	}
	if tp == service.TTSProviderKokoro && kokoroURL == "" {
		return fmt.Errorf("kokoro_base_url obrigatório para Kokoro (ou defina KOKORO_DEFAULT_BASE_URL no servidor)")
	}
	return nil
}

// HandlePatchAgent PATCH /api/v1/agents/:id
func HandlePatchAgent(log *zap.Logger, db *gorm.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_id", "id inválido", nil)
		}
		var agent model.AIAgent
		if err := db.Where("id = ? AND workspace_id = ?", id, wid).First(&agent).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "agente não encontrado", nil)
		}
		var body patchAgentBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "", nil)
		}
		updates := map[string]interface{}{}
		if body.Name != nil {
			n := strings.TrimSpace(*body.Name)
			if n == "" {
				return JSONError(c, fiber.StatusBadRequest, "validation_error", "name não pode ser vazio", nil)
			}
			updates["name"] = n
		}
		if body.Provider != nil {
			p := strings.ToLower(strings.TrimSpace(*body.Provider))
			if p != "gemini" && p != "openai" {
				return JSONError(c, fiber.StatusBadRequest, "validation_error", "provider deve ser gemini ou openai", nil)
			}
			updates["provider"] = p
		}
		if body.Model != nil {
			m := strings.TrimSpace(*body.Model)
			if m == "" {
				return JSONError(c, fiber.StatusBadRequest, "validation_error", "model não pode ser vazio", nil)
			}
			updates["model"] = m
		}
		if body.Role != nil {
			updates["role"] = strings.TrimSpace(*body.Role)
		}
		if body.Description != nil {
			updates["description"] = strings.TrimSpace(*body.Description)
		}
		if body.Active != nil {
			updates["active"] = *body.Active
		}
		if body.APIKey != nil {
			k := strings.TrimSpace(*body.APIKey)
			if k != "" {
				if strings.TrimSpace(cfg.AppEncryptionKey) == "" {
					return JSONError(c, fiber.StatusServiceUnavailable, "encryption_not_configured",
						"defina APP_ENCRYPTION_KEY no servidor para atualizar a chave API", nil)
				}
				cipher, err := cryptoagent.Encrypt(k, cfg.AppEncryptionKey)
				if err != nil {
					return JSONError(c, fiber.StatusInternalServerError, "encrypt_error", err.Error(), nil)
				}
				updates["api_key_cipher"] = cipher
				updates["api_key_last4"] = last4FromKey(k)
			}
		}
		if body.VoiceReplyEnabled != nil {
			updates["voice_reply_enabled"] = *body.VoiceReplyEnabled
		}
		if body.TTSProvider != nil {
			updates["tts_provider"] = service.NormalizeTTSProvider(*body.TTSProvider)
		}
		if body.OpenAITTSVoice != nil {
			mergedTTS := agent.TTSProvider
			if body.TTSProvider != nil {
				mergedTTS = service.NormalizeTTSProvider(*body.TTSProvider)
			}
			updates["openai_tts_voice"] = service.ResolveStoredOpenAITTSVoice(mergedTTS, *body.OpenAITTSVoice, cfg)
		}
		if body.OmnivoiceBaseURL != nil {
			updates["omnivoice_base_url"] = strings.TrimSpace(*body.OmnivoiceBaseURL)
		}
		if body.OpenAITTSModel != nil {
			updates["openai_tts_model"] = strings.TrimSpace(*body.OpenAITTSModel)
		}
		if body.KokoroBaseURL != nil {
			updates["kokoro_base_url"] = strings.TrimSpace(*body.KokoroBaseURL)
		}
		if body.OpenAITTSAPIKey != nil {
			k := strings.TrimSpace(*body.OpenAITTSAPIKey)
			if k != "" {
				if strings.TrimSpace(cfg.AppEncryptionKey) == "" {
					return JSONError(c, fiber.StatusServiceUnavailable, "encryption_not_configured",
						"defina APP_ENCRYPTION_KEY no servidor para atualizar a chave TTS", nil)
				}
				tc, err := cryptoagent.Encrypt(k, cfg.AppEncryptionKey)
				if err != nil {
					return JSONError(c, fiber.StatusInternalServerError, "encrypt_error", err.Error(), nil)
				}
				updates["openai_tts_api_key_cipher"] = tc
				updates["openai_tts_api_key_last4"] = last4FromKey(k)
			}
		}
		if body.ElevenLabsAPIKey != nil {
			k := strings.TrimSpace(*body.ElevenLabsAPIKey)
			if k != "" {
				if strings.TrimSpace(cfg.AppEncryptionKey) == "" {
					return JSONError(c, fiber.StatusServiceUnavailable, "encryption_not_configured",
						"defina APP_ENCRYPTION_KEY no servidor para atualizar a chave ElevenLabs", nil)
				}
				ec, err := cryptoagent.Encrypt(k, cfg.AppEncryptionKey)
				if err != nil {
					return JSONError(c, fiber.StatusInternalServerError, "encrypt_error", err.Error(), nil)
				}
				updates["elevenlabs_api_key_cipher"] = ec
				updates["elevenlabs_api_key_last4"] = last4FromKey(k)
			}
		}
		if err := validateAgentTTSAfterPatch(agent, body, cfg); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", err.Error(), nil)
		}
		whatsappOn := false
		if body.UseForWhatsAppAutoReply != nil {
			whatsappOn = *body.UseForWhatsAppAutoReply
			updates["use_for_whatsapp_auto_reply"] = whatsappOn
		}
		needVoicePreview := service.VoicePreviewNeedsRegenerate(updates)
		if err := db.Transaction(func(tx *gorm.DB) error {
			if body.UseForWhatsAppAutoReply != nil && whatsappOn {
				if err := service.ClearOtherWhatsAppAutoReplyAgents(tx, wid, id); err != nil {
					return err
				}
			}
			if len(updates) == 0 {
				return nil
			}
			return tx.Model(&model.AIAgent{}).Where("id = ? AND workspace_id = ?", id, wid).Updates(updates).Error
		}); err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		if err := db.Where("id = ? AND workspace_id = ?", id, wid).First(&agent).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		ttsActive := agent.VoiceReplyEnabled && service.NormalizeTTSProvider(agent.TTSProvider) != service.TTSProviderNone
		missingVoicePreview := ttsActive && strings.TrimSpace(agent.VoicePreviewPath) == ""
		if needVoicePreview || missingVoicePreview {
			ctxPrev, cancelPrev := context.WithTimeout(c.Context(), 2*time.Minute)
			defer cancelPrev()
			if err := service.RegenerateAgentVoicePreview(ctxPrev, log, db, cfg, cfg.AppEncryptionKey, &agent); err != nil && log != nil {
				log.Warn("prévia voz (patch)", zap.Error(err))
			}
			_ = db.Where("id = ? AND workspace_id = ?", id, wid).First(&agent).Error
		}
		return JSONSuccess(c, agentToDTO(agent, cfg))
	}
}

// HandleGetAgentVoicePreview GET /api/v1/agents/:id/voice-preview — áudio da última prévia gerada (Bearer).
func HandleGetAgentVoicePreview(db *gorm.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_id", "id inválido", nil)
		}
		var agent model.AIAgent
		if err := db.Where("id = ? AND workspace_id = ?", id, wid).First(&agent).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "agente não encontrado", nil)
		}
		rel := strings.TrimSpace(agent.VoicePreviewPath)
		if rel == "" {
			return JSONError(c, fiber.StatusNotFound, "not_found", "prévia de voz ainda não disponível", nil)
		}
		abs, err := service.ResolvePersistentMediaPath(cfg.MediaPersistentDir, rel)
		if err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "ficheiro em falta", nil)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "ficheiro em falta", nil)
		}
		switch strings.ToLower(filepath.Ext(abs)) {
		case ".mp3":
			c.Type("audio/mpeg")
		case ".wav":
			c.Type("audio/wav")
		default:
			c.Type("application/octet-stream")
		}
		return c.Send(data)
	}
}

// HandleDeleteAgent DELETE /api/v1/agents/:id
func HandleDeleteAgent(db *gorm.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_id", "id inválido", nil)
		}
		var existing model.AIAgent
		if err := db.Where("id = ? AND workspace_id = ?", id, wid).First(&existing).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "agente não encontrado", nil)
		}
		_ = service.RemoveStoredVoicePreview(cfg, existing.VoicePreviewPath)
		res := db.Where("id = ? AND workspace_id = ?", id, wid).Delete(&model.AIAgent{})
		if res.Error != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", res.Error.Error(), nil)
		}
		if res.RowsAffected == 0 {
			return JSONError(c, fiber.StatusNotFound, "not_found", "agente não encontrado", nil)
		}
		return JSONSuccess(c, fiber.Map{"deleted": true, "id": id.String()})
	}
}

type testAgentBody struct {
	Message string `json:"message"`
}

// HandleTestAgent POST /api/v1/agents/:id/test
func HandleTestAgent(db *gorm.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_id", "id inválido", nil)
		}
		if strings.TrimSpace(cfg.AppEncryptionKey) == "" {
			return JSONError(c, fiber.StatusServiceUnavailable, "encryption_not_configured",
				"APP_ENCRYPTION_KEY necessária para testar agente", nil)
		}
		var body testAgentBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "", nil)
		}
		msg := strings.TrimSpace(body.Message)
		if msg == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "message obrigatório", nil)
		}
		if utf8.RuneCountInString(msg) > 2000 {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "message demasiado longo (máx. 2000 caracteres)", nil)
		}
		var agent model.AIAgent
		if err := db.Where("id = ? AND workspace_id = ?", id, wid).First(&agent).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "agente não encontrado", nil)
		}
		if !agent.Active {
			return JSONError(c, fiber.StatusBadRequest, "inactive", "agente inativo", nil)
		}
		llm, err := service.BuildLLMFromAgent(cfg.AppEncryptionKey, &agent)
		if err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "llm_error", err.Error(), nil)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 90*time.Second)
		defer cancel()
		reply, err := llm.Reply(ctx, msg)
		if err != nil {
			return JSONError(c, fiber.StatusBadGateway, "llm_upstream", err.Error(), nil)
		}
		reply = service.SanitizeLLMTextForWhatsApp(reply)
		return JSONSuccess(c, fiber.Map{"reply": reply})
	}
}
