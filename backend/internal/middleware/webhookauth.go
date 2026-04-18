package middleware

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/model"
	"wa-saas/backend/internal/pkg/securestring"
)

func webhookErr(c *fiber.Ctx, status int, code, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": fiber.Map{"code": code, "message": msg}})
}

const headerWebhookSecret = "X-Webhook-Secret"

func extractWebhookAPIKey(c *fiber.Ctx) string {
	if k := strings.TrimSpace(c.Get("apikey")); k != "" {
		return k
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(c.Body(), &root) != nil {
		return ""
	}
	if raw, ok := root["apikey"]; ok {
		var key string
		_ = json.Unmarshal(raw, &key)
		return strings.TrimSpace(key)
	}
	return ""
}

// webhookAPIKeyValid aceita EVOLUTION_WEBHOOK_API_KEY, EVOLUTION_API_KEY (global) ou token da instância na BD
// (Evolution Go costuma enviar o token da instância no webhook).
func webhookAPIKeyValid(key string, cfg *config.Config, db *gorm.DB, instanceSlug string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	if cfg.EvolutionWebhookAPIKey != "" && securestring.Equal(key, cfg.EvolutionWebhookAPIKey) {
		return true
	}
	if cfg.EvolutionAPIKey != "" && securestring.Equal(key, cfg.EvolutionAPIKey) {
		return true
	}
	if db != nil {
		slug := strings.TrimSpace(strings.ToLower(instanceSlug))
		if slug != "" {
			var inst model.WhatsAppInstance
			if err := db.Where("evolution_instance_name = ?", slug).First(&inst).Error; err == nil {
				tok := strings.TrimSpace(inst.EvolutionInstanceToken)
				if tok != "" && securestring.Equal(key, tok) {
					return true
				}
			}
		}
	}
	return false
}

// WebhookAuth valida origem do webhook (header e/ou apikey no corpo Evolution).
func WebhookAuth(cfg *config.Config, db *gorm.DB, log *zap.Logger) fiber.Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return func(c *fiber.Ctx) error {
		if cfg.InsecureSkipWebhookAuth && cfg.Env == "development" {
			return c.Next()
		}

		inst := strings.TrimSpace(c.Params("instance_id"))

		okSecret := false
		if cfg.WebhookSharedSecret != "" {
			got := c.Get(headerWebhookSecret)
			okSecret = securestring.Equal(strings.TrimSpace(got), cfg.WebhookSharedSecret)
		}

		gotKey := extractWebhookAPIKey(c)
		okAPIKey := webhookAPIKeyValid(gotKey, cfg, db, inst)

		switch {
		case cfg.WebhookSharedSecret != "" && cfg.EvolutionWebhookAPIKey != "":
			if !okSecret && !okAPIKey {
				log.Warn("webhook auth rejeitado",
					zap.String("instance_path", inst),
					zap.Bool("x_webhook_secret_ok", okSecret),
					zap.Bool("apikey_ok", okAPIKey),
					zap.Bool("apikey_header_present", strings.TrimSpace(c.Get("apikey")) != ""),
					zap.Int("body_len", len(c.Body())),
				)
				return webhookErr(c, fiber.StatusUnauthorized, "webhook_unauthorized", "X-Webhook-Secret ou apikey inválidos")
			}
		case cfg.WebhookSharedSecret != "":
			if !okSecret {
				log.Warn("webhook auth rejeitado", zap.String("instance_path", inst), zap.String("motivo", "x_webhook_secret"))
				return webhookErr(c, fiber.StatusUnauthorized, "webhook_unauthorized", "X-Webhook-Secret inválido")
			}
		case cfg.EvolutionWebhookAPIKey != "":
			if !okAPIKey {
				log.Warn("webhook auth rejeitado",
					zap.String("instance_path", inst),
					zap.String("motivo", "apikey"),
					zap.Bool("apikey_header_present", strings.TrimSpace(c.Get("apikey")) != ""),
					zap.Int("body_len", len(c.Body())),
				)
				return webhookErr(c, fiber.StatusUnauthorized, "webhook_unauthorized", "apikey do corpo inválida")
			}
		default:
			log.Error("webhook sem método de autenticação configurado", zap.String("instance_path", inst))
			return webhookErr(c, fiber.StatusInternalServerError, "config_error", "webhook sem método de autenticação")
		}

		return c.Next()
	}
}
