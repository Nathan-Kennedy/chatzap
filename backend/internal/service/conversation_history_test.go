package service

import (
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

func TestBuildWhatsAppHistoryForLLM_skipsCurrentAndFormats(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Conversation{}, &model.Message{}); err != nil {
		t.Fatal(err)
	}
	cid := uuid.New()
	wid := uuid.New()
	iid := uuid.New()
	now := time.Now().UTC()
	if err := db.Create(&model.Conversation{
		ID: cid, WorkspaceID: wid, WhatsAppInstanceID: iid, ContactJID: "5511999999999@s.whatsapp.net",
		ContactName: "Nathan Kennedy", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	t1 := now.Add(-10 * time.Minute)
	t2 := now.Add(-5 * time.Minute)
	if err := db.Create(&model.Message{
		ID: uuid.New(), ConversationID: cid, Direction: "inbound", Body: "Sou o Nathan",
		MessageType: "text", CreatedAt: t1,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Message{
		ID: uuid.New(), ConversationID: cid, Direction: "outbound", Body: "Prazer!",
		MessageType: "text", CreatedAt: t2,
	}).Error; err != nil {
		t.Fatal(err)
	}
	cur := "gostaria de saber se vc lembra"
	if err := db.Create(&model.Message{
		ID: uuid.New(), ConversationID: cid, Direction: "inbound", Body: cur,
		MessageType: "text", CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	s, err := BuildWhatsAppHistoryForLLM(db, cid, cur, 20, 8000, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if s == "" {
		t.Fatal("vazio")
	}
	if !strings.Contains(s, "Nathan Kennedy") || !strings.Contains(s, "Sou o Nathan") || !strings.Contains(s, "Prazer") {
		t.Fatalf("falta contexto: %s", s)
	}
	if strings.Contains(s, cur) {
		t.Fatalf("não devia repetir a mensagem atual no histórico: %s", s)
	}
}

func TestBuildWhatsAppHistoryForLLM_excludesByMessageID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Conversation{}, &model.Message{}); err != nil {
		t.Fatal(err)
	}
	cid := uuid.New()
	wid := uuid.New()
	iid := uuid.New()
	now := time.Now().UTC()
	if err := db.Create(&model.Conversation{
		ID: cid, WorkspaceID: wid, WhatsAppInstanceID: iid, ContactJID: "5511999999999@s.whatsapp.net",
		ContactName: "Test", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	audioID := uuid.New()
	if err := db.Create(&model.Message{
		ID: audioID, ConversationID: cid, Direction: "inbound", Body: "[áudio]",
		MessageType: "audio", CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	transcribed := "[Mensagem de voz] Olá"
	s, err := BuildWhatsAppHistoryForLLM(db, cid, transcribed, 20, 8000, audioID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(s, "[áudio]") {
		t.Fatalf("histórico não devia incluir placeholder de áudio quando excluímos por ID: %s", s)
	}
}
