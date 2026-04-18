package handler

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/service"
)

type SendBody struct {
	Number   string `json:"number"`
	Text     string `json:"text"`
	Instance string `json:"instance"`
}

// HandleEvolutionSend POST /api/v1/internal/evolution/send — uso local protegido por INTERNAL_API_KEY.
func HandleEvolutionSend(log *zap.Logger, cfg *config.Config, ev *service.EvolutionClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if ev == nil {
			return JSONError(c, fiber.StatusNotImplemented, "evolution_not_configured",
				"Envio REST só com WHATSAPP_PROVIDER=evolution e EVOLUTION_* definidos", nil)
		}
		var body SendBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		if body.Number == "" || body.Text == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "number e text são obrigatórios", nil)
		}
		inst := body.Instance
		if inst == "" {
			inst = cfg.EvolutionInstanceName
		}
		if _, err := ev.SendText(c.Context(), inst, body.Number, body.Text); err != nil {
			log.Error("evolution send manual", zap.Error(err))
			return JSONError(c, fiber.StatusBadGateway, "evolution_error", err.Error(), nil)
		}
		return JSONSuccess(c, fiber.Map{"sent": true, "instance": inst})
	}
}
