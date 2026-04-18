package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/model"
)

func TestHandleWhatsAppWebhook_usesPathSlugWhenBodyInstanceIsUUID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Workspace{}, &model.WhatsAppInstance{}, &model.Conversation{}, &model.Message{}, &model.WebhookMessage{}); err != nil {
		t.Fatal(err)
	}

	wsID := uuid.New()
	now := time.Now()
	if err := db.Create(&model.Workspace{ID: wsID, Name: "t", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.WhatsAppInstance{
		ID:                     uuid.New(),
		WorkspaceID:            wsID,
		EvolutionInstanceName:  "minha-loja",
		EvolutionInstanceToken: "tok",
		DisplayName:            "Loja",
		Status:                 "connected",
		CreatedAt:              now,
		UpdatedAt:              now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	app.Post("/webhooks/whatsapp/:instance_id", HandleWhatsAppWebhook(WebhookDeps{
		Log: zap.NewNop(),
		DB:  db,
		Cfg: &config.Config{},
	}))

	body := map[string]interface{}{
		"event":    "messages.upsert",
		"instance": "f363a211-d674-4c81-9085-2d9ef051538d",
		"data": map[string]interface{}{
			"key": map[string]interface{}{
				"remoteJid": "5511999999999@s.whatsapp.net",
				"fromMe":    false,
				"id":        "msgtest1",
			},
			"message": map[string]interface{}{
				"conversation": "Olá da rua",
			},
			"messageTimestamp": float64(now.Unix()),
		},
	}
	raw, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/whatsapp/minha-loja", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, string(b))
	}

	var n int64
	if err := db.Model(&model.Message{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 message in inbox, got %d", n)
	}
}
