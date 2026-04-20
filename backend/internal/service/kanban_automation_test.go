package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

func TestApplyKanbanAutomationFromInbound_movesOnKeyword(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Conversation{}, &model.KanbanAutomationRule{}); err != nil {
		t.Fatal(err)
	}
	ws := uuid.New()
	inst := uuid.New()
	cid := uuid.New()
	conv := model.Conversation{
		ID:                 cid,
		WorkspaceID:        ws,
		WhatsAppInstanceID: inst,
		ContactJID:         "5511999999999@s.whatsapp.net",
		PipelineStage:      "novo",
	}
	if err := db.Create(&conv).Error; err != nil {
		t.Fatal(err)
	}
	rule := model.KanbanAutomationRule{
		WorkspaceID: ws,
		FromStage:   "*",
		ToStage:     "qualificado",
		Keyword:     "interesse",
		Enabled:     true,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	if err := ApplyKanbanAutomationFromInbound(db, ws, cid, "tenho interesse no serviço"); err != nil {
		t.Fatal(err)
	}
	var got model.Conversation
	if err := db.Where("id = ?", cid).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.PipelineStage != "qualificado" {
		t.Fatalf("pipeline_stage=%q want qualificado", got.PipelineStage)
	}
}

func TestApplyKanbanAutomationFromInbound_respectsFromStage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Conversation{}, &model.KanbanAutomationRule{}); err != nil {
		t.Fatal(err)
	}
	ws := uuid.New()
	inst := uuid.New()
	cid := uuid.New()
	conv := model.Conversation{
		ID:                 cid,
		WorkspaceID:        ws,
		WhatsAppInstanceID: inst,
		ContactJID:         "5511888888888@s.whatsapp.net",
		PipelineStage:      "novo",
	}
	if err := db.Create(&conv).Error; err != nil {
		t.Fatal(err)
	}
	rule := model.KanbanAutomationRule{
		WorkspaceID: ws,
		FromStage:   "qualificado",
		ToStage:     "proposta",
		Keyword:     "aceito",
		Enabled:     true,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	if err := ApplyKanbanAutomationFromInbound(db, ws, cid, "aceito a proposta"); err != nil {
		t.Fatal(err)
	}
	var got model.Conversation
	if err := db.Where("id = ?", cid).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.PipelineStage != "novo" {
		t.Fatalf("pipeline_stage=%q should stay novo when from_stage mismatch", got.PipelineStage)
	}
}
