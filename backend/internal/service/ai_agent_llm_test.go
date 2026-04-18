package service

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

func TestComposeAgentSystemPrompt_includesRoleAndDescription(t *testing.T) {
	s := ComposeAgentSystemPrompt("Clara", "Secretária", "Escritório de engenharia civil.")
	if s == "" {
		t.Fatal("vazio")
	}
	if !containsAll(s, []string{"Clara", "Secretária", "engenharia", "Brasil"}) {
		t.Fatalf("prompt incompleto: %s", s)
	}
}

func containsAll(hay string, needles []string) bool {
	for _, n := range needles {
		if !strings.Contains(hay, n) {
			return false
		}
	}
	return true
}

func TestClearOtherWhatsAppAutoReplyAgents_onlyOne(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIAgent{}); err != nil {
		t.Fatal(err)
	}
	ws := uuid.New()
	a1 := model.AIAgent{WorkspaceID: ws, Name: "A1", Provider: "gemini", Model: "m", UseForWhatsAppAutoReply: true}
	a2 := model.AIAgent{WorkspaceID: ws, Name: "A2", Provider: "gemini", Model: "m", UseForWhatsAppAutoReply: true}
	if err := db.Create(&a1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&a2).Error; err != nil {
		t.Fatal(err)
	}
	if err := ClearOtherWhatsAppAutoReplyAgents(db, ws, a2.ID); err != nil {
		t.Fatal(err)
	}
	var row1, row2 model.AIAgent
	if err := db.Where("id = ?", a1.ID).First(&row1).Error; err != nil {
		t.Fatal(err)
	}
	if row1.UseForWhatsAppAutoReply {
		t.Fatal("a1 devia estar desmarcado")
	}
	if err := db.Where("id = ?", a2.ID).First(&row2).Error; err != nil {
		t.Fatal(err)
	}
	if !row2.UseForWhatsAppAutoReply {
		t.Fatal("a2 devia permanecer marcado")
	}
}
