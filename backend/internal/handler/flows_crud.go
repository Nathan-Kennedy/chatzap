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

// HandleListFlows GET /api/v1/flows
func HandleListFlows(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		var rows []model.Flow
		if err := db.Where("workspace_id = ?", wid).Order("updated_at DESC").Find(&rows).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		out := make([]fiber.Map, 0, len(rows))
		for _, f := range rows {
			agentName := ""
			if f.AgentID != nil {
				var ag model.AIAgent
				if err := db.Where("id = ? AND workspace_id = ?", *f.AgentID, wid).First(&ag).Error; err == nil {
					agentName = ag.Name
				}
			}
			out = append(out, fiber.Map{
				"id":          f.ID.String(),
				"name":        f.Name,
				"description": f.Description,
				"agentName":   agentName,
				"published":   f.Published,
			})
		}
		return JSONSuccess(c, out)
	}
}

type createFlowBody struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	AgentID     *string `json:"agent_id"`
}

// HandleCreateFlow POST /api/v1/flows
func HandleCreateFlow(log *zap.Logger, db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		var body createFlowBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "nome é obrigatório", nil)
		}
		f := &model.Flow{
			WorkspaceID: wid,
			Name:        name,
			Description: strings.TrimSpace(body.Description),
			Published:   false,
		}
		if body.AgentID != nil && strings.TrimSpace(*body.AgentID) != "" {
			aid, err := uuid.Parse(strings.TrimSpace(*body.AgentID))
			if err != nil {
				return JSONError(c, fiber.StatusBadRequest, "validation_error", "agent_id inválido", nil)
			}
			var cnt int64
			if err := db.Model(&model.AIAgent{}).Where("id = ? AND workspace_id = ?", aid, wid).Count(&cnt).Error; err != nil {
				log.Error("create flow agent", zap.Error(err))
				return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
			}
			if cnt == 0 {
				return JSONError(c, fiber.StatusBadRequest, "validation_error", "agente não encontrado neste workspace", nil)
			}
			f.AgentID = &aid
		}
		if err := db.Create(f).Error; err != nil {
			log.Error("create flow", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		return JSONSuccess(c, fiber.Map{"id": f.ID.String()})
	}
}
