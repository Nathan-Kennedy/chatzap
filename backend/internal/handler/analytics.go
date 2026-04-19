package handler

import (
	"fmt"
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

type analyticsDayRow struct {
	Date     string `json:"date"`
	Inbound  int64  `json:"inbound"`
	Outbound int64  `json:"outbound"`
}

type analyticsHourRow struct {
	Hour     int   `json:"hour"`
	Messages int64 `json:"messages"`
}

// HandleAnalyticsTimeseries GET /api/v1/analytics/timeseries — agregações por dia e por hora (UTC), últimos 30 dias.
func HandleAnalyticsTimeseries(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wid, err := uuid.Parse(c.Locals(middleware.LocalWorkspaceID).(string))
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "", nil)
		}
		since := time.Now().UTC().AddDate(0, 0, -30)

		var dayRows []struct {
			Day      time.Time `gorm:"column:day"`
			Inbound  int64     `gorm:"column:inbound"`
			Outbound int64     `gorm:"column:outbound"`
		}
		qDay := `
SELECT DATE(m.created_at AT TIME ZONE 'UTC') AS day,
  COUNT(*) FILTER (WHERE m.direction = 'inbound') AS inbound,
  COUNT(*) FILTER (WHERE m.direction = 'outbound') AS outbound
FROM messages m
INNER JOIN conversations c ON c.id = m.conversation_id
WHERE c.workspace_id = ? AND m.created_at >= ?
GROUP BY 1
ORDER BY 1`
		if err := db.Raw(qDay, wid, since).Scan(&dayRows).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}

		byDay := make([]analyticsDayRow, 0, len(dayRows))
		for _, r := range dayRows {
			byDay = append(byDay, analyticsDayRow{
				Date:     r.Day.UTC().Format("2006-01-02"),
				Inbound:  r.Inbound,
				Outbound: r.Outbound,
			})
		}

		var hourRows []struct {
			Hour int   `gorm:"column:hr"`
			N    int64 `gorm:"column:n"`
		}
		qHour := `
SELECT EXTRACT(HOUR FROM m.created_at AT TIME ZONE 'UTC')::int AS hr,
  COUNT(*) AS n
FROM messages m
INNER JOIN conversations c ON c.id = m.conversation_id
WHERE c.workspace_id = ? AND m.created_at >= ?
GROUP BY 1
ORDER BY 1`
		if err := db.Raw(qHour, wid, since).Scan(&hourRows).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", err.Error(), nil)
		}
		hourMap := make(map[int]int64)
		for _, r := range hourRows {
			hourMap[r.Hour] = r.N
		}
		byHour := make([]analyticsHourRow, 0, 24)
		for h := 0; h < 24; h++ {
			n := hourMap[h]
			byHour = append(byHour, analyticsHourRow{Hour: h, Messages: n})
		}

		return JSONSuccess(c, fiber.Map{
			"by_day":  byDay,
			"by_hour": byHour,
			"note": fmt.Sprintf("Período: últimos 30 dias até %s (UTC).", time.Now().UTC().Format(time.RFC3339)),
		})
	}
}
