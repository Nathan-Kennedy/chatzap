package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MediaTempToken URL pública de curta duração para a Evolution Go descarregar ficheiros (POST /send/media).
type MediaTempToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Token     string    `gorm:"size:128;uniqueIndex;not null"`
	FilePath  string    `gorm:"size:1024;not null"`
	MimeType  string    `gorm:"size:128"`
	FileName  string    `gorm:"size:512"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time
}

func (MediaTempToken) TableName() string {
	return "media_temp_tokens"
}

func (m *MediaTempToken) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
