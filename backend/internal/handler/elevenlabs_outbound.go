package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/service"
)

type elevenLabsOutboundBody struct {
	ToNumber             string `json:"to_number"`
	CallRecordingEnabled *bool  `json:"call_recording_enabled"`
	RingingTimeoutSecs   *int   `json:"ringing_timeout_secs"`
}

// HandleElevenLabsOutboundCall POST /api/v1/elevenlabs/outbound-call — ligação outbound ConvAI + Twilio (ElevenLabs).
func HandleElevenLabsOutboundCall(cfg *config.Config, log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if cfg == nil {
			return JSONError(c, fiber.StatusInternalServerError, "config", "configuração em falta", nil)
		}
		key := strings.TrimSpace(cfg.ElevenLabsAPIKey)
		agentID := strings.TrimSpace(cfg.ElevenLabsConvAIAgentID)
		phoneID := strings.TrimSpace(cfg.ElevenLabsConvAIAgentPhoneNumberID)
		if key == "" || agentID == "" || phoneID == "" {
			return JSONError(c, fiber.StatusServiceUnavailable, "elevenlabs_outbound_disabled",
				"Ligações ElevenLabs não configuradas no servidor (ELEVENLABS_API_KEY, ELEVENLABS_CONVAI_AGENT_ID, ELEVENLABS_CONVAI_AGENT_PHONE_NUMBER_ID).", nil)
		}

		var body elevenLabsOutboundBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "Corpo JSON inválido.", nil)
		}
		if strings.TrimSpace(body.ToNumber) == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "Campo to_number é obrigatório (E.164, ex. +351912345678).", nil)
		}

		req := service.ElevenLabsTwilioOutboundCallRequest{
			AgentID:            agentID,
			AgentPhoneNumberID: phoneID,
			ToNumber:           body.ToNumber,
			CallRecording:      body.CallRecordingEnabled,
		}
		if body.RingingTimeoutSecs != nil && *body.RingingTimeoutSecs > 0 {
			req.TelephonyCallConfig = &service.ElevenLabsTelephonyCallConfig{
				RingingTimeoutSecs: *body.RingingTimeoutSecs,
			}
		}

		out, err := service.ElevenLabsTwilioOutboundCall(c.Context(), key, cfg.ElevenLabsAPIBaseURL, req)
		if err != nil {
			if log != nil {
				log.Warn("elevenlabs outbound call", zap.Error(err))
			}
			return JSONError(c, fiber.StatusBadGateway, "elevenlabs_outbound_failed", err.Error(), nil)
		}

		resp := fiber.Map{
			"success":         out.Success,
			"message":         out.Message,
			"conversation_id": nil,
			"call_sid":        nil,
		}
		if out.ConversationID != nil {
			resp["conversation_id"] = *out.ConversationID
		}
		if out.CallSid != nil {
			resp["call_sid"] = *out.CallSid
		}
		return JSONSuccess(c, resp)
	}
}
