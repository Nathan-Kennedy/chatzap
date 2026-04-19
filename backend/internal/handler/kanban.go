package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
)

const (
	kanbanStageNovo          = "novo"
	kanbanStageQualificado   = "qualificado"
	kanbanStageProposta      = "proposta"
	kanbanStageFechado       = "fechado"
)

func parseKanbanStage(s string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(s))
	switch v {
	case kanbanStageNovo, kanbanStageQualificado, kanbanStageProposta, kanbanStageFechado:
		return v, nil
	default:
		return "", fmt.Errorf("stage inválido (use: novo, qualificado, proposta, fechado)")
	}
}

// HandleKanbanBoard GET /api/v1/kanban/board — conversas agrupadas por pipeline_stage.
func HandleKanbanBoard(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}

		stages := []string{kanbanStageNovo, kanbanStageQualificado, kanbanStageProposta, kanbanStageFechado}
		out := fiber.Map{}
		for _, st := range stages {
			var rows []model.Conversation
			if err := db.Where("workspace_id = ? AND pipeline_stage = ?", wid, st).
				Order("last_message_at DESC NULLS LAST, updated_at DESC").
				Limit(300).
				Find(&rows).Error; err != nil {
				return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
			}
			cards := make([]fiber.Map, 0, len(rows))
			for _, r := range rows {
				phone := r.ContactJID
				if i := strings.Index(phone, "@"); i > 0 {
					phone = phone[:i]
				}
				tags := []string{}
				if strings.TrimSpace(r.Channel) != "" {
					tags = append(tags, r.Channel)
				}
				cards = append(cards, fiber.Map{
					"id":    r.ID.String(),
					"title": r.ContactName,
					"phone": phone,
					"tags":  tags,
				})
			}
			out[st] = cards
		}
		return JSONSuccess(c, fiber.Map{"stages": out})
	}
}

type kanbanMoveBody struct {
	Stage string `json:"stage"`
}

// HandleKanbanMoveCard PATCH /api/v1/kanban/cards/:conversation_id — altera pipeline_stage.
func HandleKanbanMoveCard(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		cid, err := uuid.Parse(c.Params("conversation_id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "conversation_id inválido", nil)
		}
		var body kanbanMoveBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		if strings.TrimSpace(body.Stage) == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "stage é obrigatório", nil)
		}
		stage, perr := parseKanbanStage(body.Stage)
		if perr != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", perr.Error(), nil)
		}

		var conv model.Conversation
		if err := db.Where("id = ? AND workspace_id = ?", cid, wid).First(&conv).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return JSONError(c, fiber.StatusNotFound, "not_found", "conversa não encontrada", nil)
			}
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		if err := db.Model(&conv).Updates(map[string]interface{}{
			"pipeline_stage": stage,
			"updated_at":     time.Now(),
		}).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		return JSONSuccess(c, fiber.Map{
			"id":             cid.String(),
			"pipeline_stage": stage,
		})
	}
}
