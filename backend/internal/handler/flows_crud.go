package handler

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
	"wa-saas/backend/internal/service"
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

// HandleGetFlow GET /api/v1/flows/:id
func HandleGetFlow(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		fid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}
		var f model.Flow
		if err := db.Where("id = ? AND workspace_id = ?", fid, wid).First(&f).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "fluxo não encontrado", nil)
		}
		k, err := service.ParseFlowKnowledgeJSON(f.KnowledgeJSON)
		if err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "invalid_stored_knowledge", err.Error(), nil)
		}
		preview, _ := service.FlowKnowledgePromptPreview(f.Name, f.KnowledgeJSON)
		out := fiber.Map{
			"id":             f.ID.String(),
			"name":           f.Name,
			"description":    f.Description,
			"published":      f.Published,
			"knowledge":      k,
			"prompt_preview": preview,
		}
		if f.AgentID != nil {
			out["agent_id"] = f.AgentID.String()
		} else {
			out["agent_id"] = nil
		}
		return JSONSuccess(c, out)
	}
}

type patchFlowBody struct {
	Name        *string          `json:"name"`
	Description *string          `json:"description"`
	Published   *bool            `json:"published"`
	AgentID     *string          `json:"agent_id"` // ausente = não alterar; "" = remover agente; UUID = definir
	Knowledge   *json.RawMessage `json:"knowledge"`
}

// HandlePatchFlow PATCH /api/v1/flows/:id — substituição total de `knowledge` quando enviado.
func HandlePatchFlow(log *zap.Logger, db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		fid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}
		var f model.Flow
		if err := db.Where("id = ? AND workspace_id = ?", fid, wid).First(&f).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "fluxo não encontrado", nil)
		}
		var body patchFlowBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		updates := map[string]interface{}{}
		if body.Name != nil {
			n := strings.TrimSpace(*body.Name)
			if n == "" {
				return JSONError(c, fiber.StatusBadRequest, "validation_error", "nome não pode ser vazio", nil)
			}
			updates["name"] = n
		}
		if body.Description != nil {
			updates["description"] = strings.TrimSpace(*body.Description)
		}
		if body.Published != nil {
			updates["published"] = *body.Published
		}
		if body.AgentID != nil {
			s := strings.TrimSpace(*body.AgentID)
			if s == "" {
				updates["agent_id"] = nil
			} else {
				aid, err := uuid.Parse(s)
				if err != nil {
					return JSONError(c, fiber.StatusBadRequest, "validation_error", "agent_id inválido", nil)
				}
				var cnt int64
				if err := db.Model(&model.AIAgent{}).Where("id = ? AND workspace_id = ?", aid, wid).Count(&cnt).Error; err != nil {
					log.Error("patch flow agent", zap.Error(err))
					return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
				}
				if cnt == 0 {
					return JSONError(c, fiber.StatusBadRequest, "validation_error", "agente não encontrado neste workspace", nil)
				}
				updates["agent_id"] = aid
			}
		}
		if body.Knowledge != nil {
			raw := strings.TrimSpace(string(*body.Knowledge))
			if raw == "" || raw == "null" {
				updates["knowledge_json"] = "{}"
			} else {
				if _, err := service.ParseFlowKnowledgeJSON(raw); err != nil {
					return JSONError(c, fiber.StatusBadRequest, "validation_error", err.Error(), nil)
				}
				updates["knowledge_json"] = raw
			}
		}
		if len(updates) == 0 {
			return JSONSuccess(c, fiber.Map{"ok": true})
		}
		if err := db.Model(&f).Updates(updates).Error; err != nil {
			log.Error("patch flow", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		return JSONSuccess(c, fiber.Map{"ok": true})
	}
}

// HandleDeleteFlow DELETE /api/v1/flows/:id
func HandleDeleteFlow(log *zap.Logger, db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		fid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}
		res := db.Where("id = ? AND workspace_id = ?", fid, wid).Delete(&model.Flow{})
		if res.Error != nil {
			log.Error("delete flow", zap.Error(res.Error))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", res.Error.Error(), nil)
		}
		if res.RowsAffected == 0 {
			return JSONError(c, fiber.StatusNotFound, "not_found", "fluxo não encontrado", nil)
		}
		return JSONSuccess(c, fiber.Map{"ok": true})
	}
}
