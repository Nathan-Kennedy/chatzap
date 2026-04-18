package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WhatsAppInstance liga um nome Evolution a um workspace.
type WhatsAppInstance struct {
	ID                     uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID            uuid.UUID `gorm:"type:uuid;index;not null"`
	EvolutionInstanceName  string    `gorm:"size:128;uniqueIndex;not null"`
	EvolutionInstanceToken string    `gorm:"size:191"`
	DisplayName            string    `gorm:"size:256"`
	Status                 string    `gorm:"size:32;not null"` // connected | qr_pending | disconnected
	PhoneE164              string    `gorm:"size:32"`
	MessagesToday          int       `gorm:"default:0"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (w *WhatsAppInstance) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}
