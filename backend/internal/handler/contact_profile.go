package handler

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
)

// HandleGetContactProfile GET /api/v1/conversations/:id/contact-profile
func HandleGetContactProfile(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		cid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}
		var conv model.Conversation
		if err := db.Where("id = ? AND workspace_id = ?", cid, wid).First(&conv).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "conversa não encontrada", nil)
		}
		var p model.ContactProfile
		err = db.Where("workspace_id = ? AND conversation_id = ?", wid, cid).First(&p).Error
		if err == gorm.ErrRecordNotFound {
			return JSONSuccess(c, fiber.Map{"facts": map[string]interface{}{}})
		}
		if err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		var raw map[string]interface{}
		if strings.TrimSpace(p.FactsJSON) != "" {
			_ = json.Unmarshal([]byte(p.FactsJSON), &raw)
		}
		if raw == nil {
			raw = map[string]interface{}{}
		}
		return JSONSuccess(c, fiber.Map{"facts": raw, "updated_at": p.UpdatedAt})
	}
}

type contactProfilePatchBody struct {
	Facts map[string]interface{} `json:"facts"`
}

// HandlePatchContactProfile PATCH /api/v1/conversations/:id/contact-profile
func HandlePatchContactProfile(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		cid, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "id inválido", nil)
		}
		var conv model.Conversation
		if err := db.Where("id = ? AND workspace_id = ?", cid, wid).First(&conv).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "conversa não encontrada", nil)
		}
		var body contactProfilePatchBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		if body.Facts == nil {
			body.Facts = map[string]interface{}{}
		}
		raw, err := json.Marshal(body.Facts)
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "facts inválido", nil)
		}
		var p model.ContactProfile
		err = db.Where("workspace_id = ? AND conversation_id = ?", wid, cid).First(&p).Error
		if err == gorm.ErrRecordNotFound {
			p = model.ContactProfile{
				WorkspaceID:    wid,
				ConversationID: cid,
				FactsJSON:      string(raw),
			}
			if err := db.Create(&p).Error; err != nil {
				return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
			}
			return JSONSuccess(c, fiber.Map{"ok": true})
		}
		if err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		if err := db.Model(&p).Update("facts_json", string(raw)).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		return JSONSuccess(c, fiber.Map{"ok": true})
	}
}
