package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
)

// HandleKanbanAutomationList GET /api/v1/kanban/automation-rules
func HandleKanbanAutomationList(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		var rows []model.KanbanAutomationRule
		if err := db.Where("workspace_id = ?", wid).Order("priority ASC, created_at ASC").Find(&rows).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		out := make([]fiber.Map, 0, len(rows))
		for _, r := range rows {
			out = append(out, fiber.Map{
				"id":         r.ID.String(),
				"from_stage": r.FromStage,
				"to_stage":   r.ToStage,
				"keyword":    r.Keyword,
				"enabled":    r.Enabled,
				"priority":   r.Priority,
			})
		}
		return JSONSuccess(c, fiber.Map{"rules": out})
	}
}

type kanbanAutomationCreateBody struct {
	FromStage string `json:"from_stage"`
	ToStage   string `json:"to_stage"`
	Keyword   string `json:"keyword"`
	Enabled   *bool  `json:"enabled"`
	Priority  *int   `json:"priority"`
}

// HandleKanbanAutomationCreate POST /api/v1/kanban/automation-rules
func HandleKanbanAutomationCreate(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		var body kanbanAutomationCreateBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		to, err := parseKanbanStage(body.ToStage)
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", err.Error(), nil)
		}
		kw := strings.TrimSpace(body.Keyword)
		if kw == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "keyword é obrigatório", nil)
		}
		from := strings.TrimSpace(body.FromStage)
		if from == "" {
			from = "*"
		}
		if from != "*" {
			if _, err := parseKanbanStage(from); err != nil {
				return JSONError(c, fiber.StatusBadRequest, "validation_error", "from_stage: use novo, qualificado, proposta, fechado ou *", nil)
			}
		}
		en := true
		if body.Enabled != nil {
			en = *body.Enabled
		}
		pr := 0
		if body.Priority != nil {
			pr = *body.Priority
		}
		row := model.KanbanAutomationRule{
			WorkspaceID: wid,
			FromStage:   from,
			ToStage:     to,
			Keyword:     kw,
			Enabled:     en,
			Priority:    pr,
		}
		if err := db.Create(&row).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		return JSONSuccess(c, fiber.Map{"id": row.ID.String()})
	}
}

// HandleKanbanAutomationUpdate PATCH /api/v1/kanban/automation-rules/:id
func HandleKanbanAutomationUpdate(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		rid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}
		var body kanbanAutomationCreateBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		var row model.KanbanAutomationRule
		if err := db.Where("id = ? AND workspace_id = ?", rid, wid).First(&row).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "regra não encontrada", nil)
		}
		updates := map[string]interface{}{}
		if body.ToStage != "" {
			to, err := parseKanbanStage(body.ToStage)
			if err != nil {
				return JSONError(c, fiber.StatusBadRequest, "validation_error", err.Error(), nil)
			}
			updates["to_stage"] = to
		}
		if body.Keyword != "" {
			updates["keyword"] = strings.TrimSpace(body.Keyword)
		}
		if body.FromStage != "" {
			from := strings.TrimSpace(body.FromStage)
			if from != "*" {
				if _, err := parseKanbanStage(from); err != nil {
					return JSONError(c, fiber.StatusBadRequest, "validation_error", "from_stage inválido", nil)
				}
			}
			updates["from_stage"] = from
		}
		if body.Enabled != nil {
			updates["enabled"] = *body.Enabled
		}
		if body.Priority != nil {
			updates["priority"] = *body.Priority
		}
		if len(updates) == 0 {
			return JSONSuccess(c, fiber.Map{"id": row.ID.String()})
		}
		if err := db.Model(&row).Updates(updates).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		return JSONSuccess(c, fiber.Map{"ok": true})
	}
}

// HandleKanbanAutomationDelete DELETE /api/v1/kanban/automation-rules/:id
func HandleKanbanAutomationDelete(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		rid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}
		res := db.Where("id = ? AND workspace_id = ?", rid, wid).Delete(&model.KanbanAutomationRule{})
		if res.Error != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", res.Error.Error(), nil)
		}
		if res.RowsAffected == 0 {
			return JSONError(c, fiber.StatusNotFound, "not_found", "regra não encontrada", nil)
		}
		return JSONSuccess(c, fiber.Map{"ok": true})
	}
}
