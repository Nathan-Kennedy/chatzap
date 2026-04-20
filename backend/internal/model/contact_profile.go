package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ContactProfile factos JSON por conversa (MVP CRM — preenchimento manual ou futura extração IA).
type ContactProfile struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID    uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_cp_ws_conv;not null"`
	ConversationID uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_cp_ws_conv;not null"`
	// FactsJSON: objeto JSON livre (CPF, endereço, links, etc.)
	FactsJSON string `gorm:"type:text;column:facts_json"`
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (ContactProfile) TableName() string {
	return "contact_profiles"
}

func (p *ContactProfile) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
