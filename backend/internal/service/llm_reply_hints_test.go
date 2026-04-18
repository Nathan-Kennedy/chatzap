package service

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

func TestContinuationStyleHint_afterGreetingOutbound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Conversation{}, &model.Message{}); err != nil {
		t.Fatal(err)
	}
	cid := uuid.New()
	now := time.Now().UTC()
	if err := db.Create(&model.Conversation{
		ID: cid, WorkspaceID: uuid.New(), WhatsAppInstanceID: uuid.New(),
		ContactJID: "5511999999999@s.whatsapp.net", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	t0 := now.Add(-2 * time.Minute)
	if err := db.Create(&model.Message{
		ID: uuid.New(), ConversationID: cid, Direction: "outbound",
		Body: "Olá, Nathan! Que bom.\nSegue o texto.", CreatedAt: t0,
	}).Error; err != nil {
		t.Fatal(err)
	}
	t1 := now.Add(-1 * time.Minute)
	if err := db.Create(&model.Message{
		ID: uuid.New(), ConversationID: cid, Direction: "outbound",
		Body: "Amanhã combina sim; qual horário?", CreatedAt: t1,
	}).Error; err != nil {
		t.Fatal(err)
	}
	h := ContinuationStyleHint(db, cid)
	if h == "" {
		t.Fatal("esperava hint após bolha com Olá na primeira linha")
	}
}

func TestContinuationStyleHint_noOutbound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Conversation{}, &model.Message{}); err != nil {
		t.Fatal(err)
	}
	cid := uuid.New()
	now := time.Now().UTC()
	if err := db.Create(&model.Conversation{
		ID: cid, WorkspaceID: uuid.New(), WhatsAppInstanceID: uuid.New(),
		ContactJID: "5511999999999@s.whatsapp.net", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if ContinuationStyleHint(db, cid) != "" {
		t.Fatal("sem outbound não deve haver hint")
	}
}

func TestStripLeadingSalutationNameLine(t *testing.T) {
	in := "Olá, Nathanzinho! 😊\nAmanhã pode ser sim."
	want := "Amanhã pode ser sim."
	if got := StripLeadingSalutationNameLine(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := StripLeadingSalutationNameLine("Oi, tudo certo?\nSim"); got != "Oi, tudo certo?\nSim" {
		t.Fatalf("frase de cortesia após Oi, não cortar: %q", got)
	}
	if got := StripLeadingSalutationNameLine("Oi, Maria!\nSegue."); got != "Segue." {
		t.Fatalf("Oi, Nome ainda pode ser cortado: %q", got)
	}
	if StripLeadingSalutationNameLine("só uma linha") != "só uma linha" {
		t.Fatal("uma linha só mantém")
	}
}

func TestRecentAssistantLineLooksLikeGreeting(t *testing.T) {
	if !recentAssistantLineLooksLikeGreeting("Olá!\nMais") {
		t.Fatal("primeira linha Olá")
	}
	if recentAssistantLineLooksLikeGreeting("Perfeito, amanhã às 10.") {
		t.Fatal("não é saudação")
	}
}
