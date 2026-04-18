package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
)

// HandleAnalyticsOverview GET /api/v1/analytics/overview
func HandleAnalyticsOverview(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}

		since := time.Now().UTC().AddDate(0, 0, -30)

		var msgCount int64
		_ = db.Table("messages").
			Joins("INNER JOIN conversations ON conversations.id = messages.conversation_id").
			Where("conversations.workspace_id = ? AND messages.created_at >= ?", wid, since).
			Count(&msgCount).Error

		var convCount int64
		_ = db.Model(&model.Conversation{}).Where("workspace_id = ?", wid).Count(&convCount).Error

		var instCount int64
		_ = db.Model(&model.WhatsAppInstance{}).Where("workspace_id = ?", wid).Count(&instCount).Error

		return JSONSuccess(c, fiber.Map{
			"messages_last_30d": msgCount,
			"conversations_total": convCount,
			"instances_total":     instCount,
		})
	}
}
