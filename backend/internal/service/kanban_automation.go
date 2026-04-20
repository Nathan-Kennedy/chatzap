package service

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

// ApplyKanbanAutomationFromInbound avalia regras do workspace e actualiza pipeline_stage se corresponder.
func ApplyKanbanAutomationFromInbound(db *gorm.DB, workspaceID, conversationID uuid.UUID, inboundTextLower string) error {
	if db == nil || workspaceID == uuid.Nil || conversationID == uuid.Nil {
		return nil
	}
	inboundTextLower = strings.TrimSpace(strings.ToLower(inboundTextLower))
	if inboundTextLower == "" {
		return nil
	}

	var conv model.Conversation
	if err := db.Where("id = ? AND workspace_id = ?", conversationID, workspaceID).First(&conv).Error; err != nil {
		return err
	}
	cur := strings.ToLower(strings.TrimSpace(conv.PipelineStage))
	if cur == "" {
		cur = "novo"
	}

	var rules []model.KanbanAutomationRule
	if err := db.Where("workspace_id = ? AND enabled = ?", workspaceID, true).
		Order("priority ASC, created_at ASC").
		Find(&rules).Error; err != nil {
		return err
	}

	for _, r := range rules {
		kw := strings.ToLower(strings.TrimSpace(r.Keyword))
		if kw == "" {
			continue
		}
		if !strings.Contains(inboundTextLower, kw) {
			continue
		}
		from := strings.TrimSpace(strings.ToLower(r.FromStage))
		if from != "" && from != "*" && from != cur {
			continue
		}
		to := strings.TrimSpace(strings.ToLower(r.ToStage))
		if to == "" || to == cur {
			continue
		}
		return db.Model(&model.Conversation{}).
			Where("id = ? AND workspace_id = ?", conversationID, workspaceID).
			Updates(map[string]interface{}{
				"pipeline_stage": to,
				"updated_at":     time.Now(),
			}).Error
	}
	return nil
}
