package service

import (
	"context"
	"strings"
	"testing"

	"wa-saas/backend/internal/config"
)

func TestGravableFollowUpText_heuristicNoGemini(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}
	out, err := GravableFollowUpText(ctx, cfg, "Visita técnica na segunda às 14h. Orçamento aproximado R$ 800,00.")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Resumo para registo") {
		t.Fatalf("expected header, got %q", out)
	}
	if !strings.Contains(strings.ToLower(out), "segunda") && !strings.Contains(out, "800") {
		t.Fatalf("expected factual line, got %q", out)
	}
}

func TestGravableFollowUpText_emptyError(t *testing.T) {
	_, err := GravableFollowUpText(context.Background(), &config.Config{}, "   ")
	if err == nil {
		t.Fatal("expected error for empty")
	}
}
