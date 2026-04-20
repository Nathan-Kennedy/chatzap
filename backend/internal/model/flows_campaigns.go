package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Flow modelo de fluxo (MVP — sem motor de execução).
type Flow struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID  `gorm:"type:uuid;index;not null"`
	Name        string     `gorm:"size:256;not null"`
	Description string     `gorm:"type:text"`
	AgentID     *uuid.UUID `gorm:"type:uuid"`
	Published   bool       `gorm:"default:false"`
	// KnowledgeJSON: produtos, serviços, horários, links, imagens (URLs), notas — ver FlowKnowledge.
	KnowledgeJSON string `gorm:"type:text;column:knowledge_json"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (f *Flow) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

// Campaign campanha em rascunho (MVP — sem envio em massa).
type Campaign struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;index;not null"`
	Name        string    `gorm:"size:256;not null"`
	Channel     string    `gorm:"size:32;default:whatsapp"`
	Status      string    `gorm:"size:32;default:draft"`
	Sent        int       `gorm:"default:0"`
	Delivered   int       `gorm:"default:0"`
	ReadCount   int       `gorm:"column:read_count;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (c *Campaign) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
