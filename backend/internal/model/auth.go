package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Workspace tenant.
type Workspace struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"size:256;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (w *Workspace) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// User credenciais locais (MVP).
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email        string    `gorm:"size:320;uniqueIndex;not null"`
	PasswordHash string    `gorm:"size:256;not null"`
	Name         string    `gorm:"size:256"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// WorkspaceMember liga utilizador a workspace com papel RBAC.
type WorkspaceMember struct {
	WorkspaceID uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	Role        string    `gorm:"size:32;not null"` // admin | supervisor | agent
	CreatedAt   time.Time
}

// RefreshToken sessão refresh (hash do token bruto).
type RefreshToken struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID `gorm:"type:uuid;index;not null"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null"`
	TokenHash   string    `gorm:"size:64;uniqueIndex;not null"`
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
