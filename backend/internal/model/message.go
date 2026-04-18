package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WebhookMessage registra eventos recebidos (auditoria MVP).
type WebhookMessage struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	InstanceID string    `gorm:"size:128;index"`
	Event      string    `gorm:"size:64;index"`
	RemoteJID  string    `gorm:"size:256;index"`
	Direction  string    `gorm:"size:16"` // inbound | outbound | event
	Body       string    `gorm:"type:text"`
	RawPayload []byte    `gorm:"type:jsonb"`
	CreatedAt  time.Time
}

func (WebhookMessage) TableName() string {
	return "webhook_messages"
}

func (m *WebhookMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
