package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// KanbanAutomationRule regra automática: ao receber mensagem inbound com palavra-chave, mover pipeline_stage.
type KanbanAutomationRule struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;index;not null"`
	// FromStage: estágio actual da conversa para aplicar; "*" = qualquer
	FromStage string `gorm:"size:24;not null;default:*"`
	ToStage   string `gorm:"size:24;not null"`
	// Keyword: substring case-insensitive no texto inbound
	Keyword   string `gorm:"size:256;not null"`
	Enabled   bool   `gorm:"default:true"`
	Priority  int    `gorm:"default:0"` // menor = avaliado primeiro
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (KanbanAutomationRule) TableName() string {
	return "kanban_automation_rules"
}

func (r *KanbanAutomationRule) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
