package handler

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
	"wa-saas/backend/internal/service"
)

// HandlePublicMediaTemp GET /media/temp/:token — sem auth (Evolution Go descarrega antes do /send/media completar).
func HandlePublicMediaTemp(db *gorm.DB, log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Params("token")
		if token == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "token em falta", nil)
		}
		var row model.MediaTempToken
		if err := db.Where("token = ?", token).First(&row).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return JSONError(c, fiber.StatusNotFound, "not_found", "ficheiro expirado ou inexistente", nil)
			}
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		if time.Now().UTC().After(row.ExpiresAt) {
			service.DeleteMediaTempToken(db, &row)
			return JSONError(c, fiber.StatusNotFound, "not_found", "ficheiro expirado", nil)
		}
		data, err := os.ReadFile(row.FilePath)
		if err != nil {
			if log != nil {
				preview := token
				if len(preview) > 12 {
					preview = preview[:12] + "…"
				}
				log.Warn("media temp read", zap.String("token", preview), zap.Error(err))
			}
			return JSONError(c, fiber.StatusNotFound, "not_found", "ficheiro indisponível", nil)
		}
		ct := row.MimeType
		if ct == "" {
			ct = "application/octet-stream"
		}
		c.Set("Content-Type", ct)
		c.Set("Cache-Control", "no-store")
		return c.Send(data)
	}
}
