package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
)

type patchWorkspaceBody struct {
	Name string `json:"name"`
}

// HandleGetWorkspace GET /api/v1/workspace
func HandleGetWorkspace(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		var ws model.Workspace
		if err := db.First(&ws, "id = ?", wid).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "", nil)
		}
		return JSONSuccess(c, fiber.Map{
			"id":   ws.ID.String(),
			"name": ws.Name,
		})
	}
}

// HandlePatchWorkspace PATCH /api/v1/workspace
func HandlePatchWorkspace(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		var body patchWorkspaceBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "", nil)
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "name obrigatório", nil)
		}
		if err := db.Model(&model.Workspace{}).Where("id = ?", wid).Update("name", name).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		return JSONSuccess(c, fiber.Map{"id": wid.String(), "name": name})
	}
}
