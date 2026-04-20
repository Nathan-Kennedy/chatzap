package middleware

import (
	"bytes"
	"encoding/json"
	"io"
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

// Garante que WebhookAuth registado na rota POST (não no Group) vê :instance_id e aceita instanceToken no JSON (Evolution Go).
func TestWebhookAuth_instanceTokenWithPathSlug(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Workspace{}, &model.WhatsAppInstance{}); err != nil {
		t.Fatal(err)
	}
	wsID := uuid.New()
	now := time.Now()
	if err := db.Create(&model.Workspace{ID: wsID, Name: "w", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.WhatsAppInstance{
		ID:                     uuid.New(),
		WorkspaceID:            wsID,
		EvolutionInstanceName:  "cuiudo_construcoes",
		EvolutionInstanceToken: "inst-secret-xyz",
		DisplayName:            "x",
		Status:                 "connected",
		CreatedAt:              now,
		UpdatedAt:              now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Env:                     "production",
		InsecureSkipWebhookAuth: false,
		WebhookSharedSecret:     "shared",
		EvolutionWebhookAPIKey:  "global-evo",
	}

	app := fiber.New()
	wh := app.Group("/webhooks")
	wh.Post("/whatsapp/:instance_id", WebhookAuth(cfg, db, zap.NewNop()), func(c *fiber.Ctx) error {
		return c.SendStatus(204)
	})

	payload := map[string]interface{}{
		"event":         "Message",
		"instanceToken": "inst-secret-xyz",
		"instanceId":    "uuid-here",
		"data":          map[string]interface{}{},
	}
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhooks/whatsapp/cuiudo_construcoes", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 204 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d want 204: %s", resp.StatusCode, string(b))
	}
}
