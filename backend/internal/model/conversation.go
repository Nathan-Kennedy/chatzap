package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Conversation thread por contacto + instância WhatsApp.
type Conversation struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID        uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_conv_tenant_contact;not null"`
	// Coluna em Postgres: whats_app_instance_id (GORM default a partir de WhatsAppInstanceID).
	WhatsAppInstanceID uuid.UUID `gorm:"type:uuid;column:whats_app_instance_id;uniqueIndex:idx_conv_tenant_contact;not null"`
	// GORM default para ContactJID é contact_j_id (não contact_jid); tag fixa o nome da coluna na BD.
	ContactJID string `gorm:"size:256;column:contact_j_id;uniqueIndex:idx_conv_tenant_contact;not null"`
	ContactName        string    `gorm:"size:512"`
	LastMessageAt      time.Time `gorm:"index"`
	LastMessagePreview string    `gorm:"size:512"`
	Channel            string    `gorm:"size:32;default:whatsapp"`
	AssignedAgentInitials string `gorm:"size:8"`
	UpdatedAt          time.Time
	CreatedAt          time.Time
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// Message mensagem na conversa.
type Message struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	ConversationID uuid.UUID `gorm:"type:uuid;index;not null"`
	Direction      string    `gorm:"size:16;not null"` // inbound | outbound
	Body           string    `gorm:"type:text"`
	ExternalID     string    `gorm:"size:128;index"`
	// MessageType: text | image | video | audio | document (inbound pode continuar só texto no MVP).
	MessageType string `gorm:"size:32;default:text;column:message_type"`
	FileName    string `gorm:"size:512;column:file_name"`
	MimeType    string `gorm:"size:128;column:mime_type"`
	// Ficheiro persistido na API (envios pelo site) — chave relativa sob MEDIA_PERSISTENT_DIR.
	StoredMediaPath string `gorm:"size:1024;column:stored_media_path"`
	// URL remota (ex. webhook WhatsApp) — servida via proxy no GET attachment.
	MediaRemoteURL string `gorm:"type:text;column:media_remote_url"`
	// JSON do nó "message" (proto Baileys/whatsmeow) — obrigatório para Evolution Go: POST /message/downloadmedia.
	WaMediaMessageJSON string `gorm:"type:text;column:wa_media_message_json"`
	CreatedAt          time.Time `gorm:"index"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
