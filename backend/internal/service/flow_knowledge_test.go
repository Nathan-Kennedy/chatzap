package service

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

func TestFormatFlowKnowledgeForPrompt_empty(t *testing.T) {
	if s := FormatFlowKnowledgeForPrompt("X", model.FlowKnowledge{}); s != "" {
		t.Fatalf("expected empty, got %q", s)
	}
}

func TestFormatFlowKnowledgeForPrompt_produtos(t *testing.T) {
	k := model.FlowKnowledge{
		Produtos: []model.FlowProduct{
			{Nome: "Kit A", Descricao: "completo", PrecoReferencia: "R$ 100"},
		},
	}
	s := FormatFlowKnowledgeForPrompt("Vendas", k)
	if !strings.Contains(s, "## Vendas") || !strings.Contains(s, "Kit A") || !strings.Contains(s, "100") {
		t.Fatalf("unexpected: %s", s)
	}
}

func TestTruncateRunes(t *testing.T) {
	s := strings.Repeat("a", AggregatedFlowKnowledgeMaxRunes+50)
	out := truncateRunes(s, AggregatedFlowKnowledgeMaxRunes)
	if !strings.HasSuffix(out, "… [truncado]") {
		t.Fatal("expected truncation marker")
	}
}

func TestAggregatedFlowKnowledgeForAgent_onlyPublished(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Flow{}, &model.AIAgent{}); err != nil {
		t.Fatal(err)
	}
	ws := uuid.New()
	ag := uuid.New()
	a := model.AIAgent{ID: ag, WorkspaceID: ws, Name: "Bot", Provider: "gemini", Model: "m"}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	f1 := model.Flow{
		WorkspaceID: ws, Name: "P1", Published: false, AgentID: &ag,
		KnowledgeJSON: `{"produtos":[{"nome":"X","descricao":"","preco_referencia":""}]}`,
	}
	f2 := model.Flow{
		WorkspaceID: ws, Name: "P2", Published: true, AgentID: &ag,
		KnowledgeJSON: `{"produtos":[{"nome":"Y","descricao":"","preco_referencia":""}]}`,
	}
	if err := db.Create(&f1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&f2).Error; err != nil {
		t.Fatal(err)
	}
	s, err := AggregatedFlowKnowledgeForAgent(db, ws, ag)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(s, "X") || !strings.Contains(s, "Y") {
		t.Fatalf("expected only published flow knowledge, got: %s", s)
	}
}
