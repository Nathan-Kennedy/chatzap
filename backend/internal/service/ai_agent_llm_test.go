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
	s := ComposeAgentSystemPrompt("Clara", "Secretária", "Escritório de engenharia civil.", false)
	if s == "" {
		t.Fatal("vazio")
	}
	if !containsAll(s, []string{"Clara", "Secretária", "engenharia", "Brasil"}) {
		t.Fatalf("prompt incompleto: %s", s)
	}
	if strings.Contains(s, "Respostas em áudio (TTS)") {
		t.Fatal("bloco TTS não devia aparecer com voiceTTSActive=false")
	}
}

func TestComposeAgentSystemPrompt_voiceTTSIncludesHint(t *testing.T) {
	s := ComposeAgentSystemPrompt("Clara", "Secretária", "Escritório.", true)
	if !strings.Contains(s, "Respostas em áudio (TTS)") || !strings.Contains(s, "manda áudio") {
		t.Fatalf("falta instrução TTS no prompt: %s", s)
	}
	if !strings.Contains(s, "[PAUSA]") {
		t.Fatalf("falta menção às etiquetas Gemini TTS: %s", s)
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

func TestWorkspaceAutoReplyNoLLMReason_noneMarked(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIAgent{}); err != nil {
		t.Fatal(err)
	}
	ws := uuid.New()
	r := WorkspaceAutoReplyNoLLMReason(db, ws)
	if !strings.Contains(r, "use_for_whatsapp_auto_reply=true") {
		t.Fatalf("unexpected: %q", r)
	}
}

func TestWorkspaceAutoReplyNoLLMReason_inactive(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIAgent{}); err != nil {
		t.Fatal(err)
	}
	ws := uuid.New()
	a := model.AIAgent{
		ID: uuid.New(), WorkspaceID: ws, Name: "Bot", Provider: "gemini", Model: "m",
		APIKeyCipher: "x", Active: true, UseForWhatsAppAutoReply: true,
	}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.AIAgent{}).Where("id = ?", a.ID).Update("active", false).Error; err != nil {
		t.Fatal(err)
	}
	r := WorkspaceAutoReplyNoLLMReason(db, ws)
	if !strings.Contains(r, "active=false") {
		t.Fatalf("unexpected: %q", r)
	}
}

func TestWorkspaceAutoReplyNoLLMReason_noCipher(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AIAgent{}); err != nil {
		t.Fatal(err)
	}
	ws := uuid.New()
	a := model.AIAgent{
		ID: uuid.New(), WorkspaceID: ws, Name: "Bot", Provider: "gemini", Model: "m",
		APIKeyCipher: "", Active: true, UseForWhatsAppAutoReply: true,
	}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	r := WorkspaceAutoReplyNoLLMReason(db, ws)
	if !strings.Contains(r, "sem chave LLM") {
		t.Fatalf("unexpected: %q", r)
	}
}
