package handler

import (
	"strings"
	"time"
	"unicode"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
)

// HandleListContacts GET /api/v1/contacts — derivado de conversas (MVP).
func HandleListContacts(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		q := strings.TrimSpace(c.Query("search"))

		tx := db.Model(&model.Conversation{}).Where("workspace_id = ?", wid).Order("contact_name ASC").Limit(500)
		if q != "" {
			like := "%" + strings.ToLower(q) + "%"
			var digits strings.Builder
			for _, r := range q {
				if unicode.IsDigit(r) {
					digits.WriteRune(r)
				}
			}
			d := digits.String()
			if len(d) >= 3 {
				tx = tx.Where(
					"(LOWER(contact_name) LIKE ? OR LOWER(contact_j_id) LIKE ? OR regexp_replace(contact_j_id, '[^0-9]', '', 'g') LIKE ?)",
					like, like, "%"+d+"%",
				)
			} else {
				tx = tx.Where("LOWER(contact_name) LIKE ? OR LOWER(contact_j_id) LIKE ?", like, like)
			}
		}

		var rows []model.Conversation
		if err := tx.Find(&rows).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		out := make([]fiber.Map, 0, len(rows))
		for _, r := range rows {
			phone := r.ContactJID
			if i := strings.Index(phone, "@"); i > 0 {
				phone = phone[:i]
			}
			seen := ""
			if !r.LastMessageAt.IsZero() {
				seen = r.LastMessageAt.UTC().Format(time.RFC3339)
			}
			out = append(out, fiber.Map{
				"id":           r.ID.String(),
				"name":         r.ContactName,
				"phone":        phone,
				"channel":      r.Channel,
				"last_seen_at": seen,
			})
		}
		return JSONSuccess(c, out)
	}
}
