package service

import (
	"encoding/json"
	"testing"
)

func TestParseEvolutionSendTextResponse_flatKey(t *testing.T) {
	raw := []byte(`{"key":{"remoteJid":"5511999999999@s.whatsapp.net","fromMe":true,"id":"ABCOUT"},"message":{"conversation":"x"}}`)
	rj, id := ParseEvolutionSendTextResponse(raw)
	if rj != "5511999999999@s.whatsapp.net" || id != "ABCOUT" {
		t.Fatalf("got rj=%q id=%q", rj, id)
	}
}

func TestParseEvolutionSendTextResponse_messageIdTopLevel(t *testing.T) {
	raw := []byte(`{"messageId":"TOP123","data":{"key":{"remoteJid":"5511888888888@s.whatsapp.net","fromMe":true}}}`)
	rj, id := ParseEvolutionSendTextResponse(raw)
	if id != "TOP123" || rj != "5511888888888@s.whatsapp.net" {
		t.Fatalf("rj=%q id=%q", rj, id)
	}
}

func TestParseEvolutionSendTextResponse_wrappedData(t *testing.T) {
	raw := []byte(`{"data":{"Key":{"RemoteJid":"5569888888888@s.whatsapp.net","FromMe":true,"Id":"XYZ"}}}`)
	rj, id := ParseEvolutionSendTextResponse(raw)
	if rj != "5569888888888@s.whatsapp.net" || id != "XYZ" {
		t.Fatalf("got rj=%q id=%q", rj, id)
	}
}

func TestPersistPortalOutboundWebhook_reconcileParseable(t *testing.T) {
	var payload EvolutionWebhookPayload
	inner, err := json.Marshal(map[string]interface{}{
		"key": map[string]interface{}{
			"remoteJid": "5511777777777@s.whatsapp.net",
			"fromMe":    true,
			"id":        "p1",
		},
		"message":          map[string]interface{}{"conversation": "olá portal"},
		"messageTimestamp": float64(1_700_000_000),
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(struct {
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}{"messages.upsert", inner})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	data := NormalizeWebhookData(payload.Data)
	got, ok := ParseInboundFromEvolution(payload.Event, data)
	if !ok || !got.FromMe || got.Text != "olá portal" || got.KeyID != "p1" {
		t.Fatalf("ok=%v %+v", ok, got)
	}
}
