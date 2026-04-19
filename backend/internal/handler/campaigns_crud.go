package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
)

// HandleListCampaigns GET /api/v1/campaigns
func HandleListCampaigns(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		var rows []model.Campaign
		if err := db.Where("workspace_id = ?", wid).Order("updated_at DESC").Find(&rows).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		out := make([]fiber.Map, 0, len(rows))
		for _, x := range rows {
			out = append(out, fiber.Map{
				"id":        x.ID.String(),
				"name":      x.Name,
				"channel":   x.Channel,
				"status":    x.Status,
				"sent":      x.Sent,
				"delivered": x.Delivered,
				"read":      x.ReadCount,
			})
		}
		return JSONSuccess(c, out)
	}
}

type createCampaignBody struct {
	Name    string `json:"name"`
	Channel string `json:"channel"`
}

// HandleCreateCampaign POST /api/v1/campaigns — rascunho (sem envio).
func HandleCreateCampaign(log *zap.Logger, db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		var body createCampaignBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "nome é obrigatório", nil)
		}
		ch := strings.TrimSpace(body.Channel)
		if ch == "" {
			ch = "whatsapp"
		}
		x := &model.Campaign{
			WorkspaceID: wid,
			Name:        name,
			Channel:     ch,
			Status:      "draft",
		}
		if err := db.Create(x).Error; err != nil {
			log.Error("create campaign", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		return JSONSuccess(c, fiber.Map{"id": x.ID.String()})
	}
}
