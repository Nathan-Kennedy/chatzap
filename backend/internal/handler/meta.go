package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"wa-saas/backend/internal/config"
)

const APIVersion = "0.3.0-mvp"

// HandleMeta GET /api/v1/meta — versão, fornecedor LLM configurado (sem expor chaves).
func HandleMeta(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		elOutbound := strings.TrimSpace(cfg.ElevenLabsAPIKey) != "" &&
			strings.TrimSpace(cfg.ElevenLabsConvAIAgentID) != "" &&
			strings.TrimSpace(cfg.ElevenLabsConvAIAgentPhoneNumberID) != ""
		return JSONSuccess(c, fiber.Map{
			"service":                          "wa-saas-api",
			"version":                          APIVersion,
			"whatsapp_provider":                cfg.WhatsAppProvider,
			"llm_provider":                     cfg.LLMProvider,
			"auto_reply_enabled":               cfg.AutoReplyEnabled,
			"gemini_model":                     cfg.GeminiModel,
			"openai_model":                     cfg.OpenAIModel,
			"elevenlabs_outbound_call_enabled": elOutbound,
		})
	}
}
