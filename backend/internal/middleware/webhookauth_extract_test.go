package middleware

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestExtractWebhookAPIKey_instanceToken(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		got := extractWebhookAPIKey(c)
		if got != "per-instance-secret" {
			t.Fatalf("got %q", got)
		}
		return c.SendStatus(204)
	})
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"event":"Message","instanceToken":"per-instance-secret","data":{}}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 204 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, string(b))
	}
}
