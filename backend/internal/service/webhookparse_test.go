package service

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizeEpochToTime_secondsVsMillis(t *testing.T) {
	sec := NormalizeEpochToTime(1_700_000_000)
	if sec.Unix() != 1_700_000_000 {
		t.Fatalf("seconds: got %d", sec.Unix())
	}
	ms := NormalizeEpochToTime(1_700_000_000_123)
	if ms.Unix() != 1_700_000_000 {
		t.Fatalf("millis: got unix %d", ms.Unix())
	}
}

func TestParseInboundFromEvolution_conversation(t *testing.T) {
	data := json.RawMessage(`{
		"key": {"remoteJid": "5511999999999@s.whatsapp.net", "fromMe": false, "id": "ABC123"},
		"message": {"conversation": "  Olá  "}
	}`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.From != "5511999999999@s.whatsapp.net" || got.Text != "Olá" || got.FromMe || got.KeyID != "ABC123" {
		t.Fatalf("got %+v", got)
	}
}

func TestParseInboundFromEvolution_extendedText(t *testing.T) {
	data := json.RawMessage(`{
		"key": {"remoteJid": "x@c.us", "fromMe": false},
		"message": {"extendedTextMessage": {"text": "teste"}}
	}`)
	got, ok := ParseInboundFromEvolution("MESSAGES.UPSERT", data)
	if !ok || got.Text != "teste" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestParseInboundFromEvolution_fromMeParsed(t *testing.T) {
	data := json.RawMessage(`{
		"key": {"remoteJid": "5511888888888@s.whatsapp.net", "fromMe": true, "id": "out1"},
		"message": {"conversation": "eu enviei"}
	}`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok || !got.FromMe || got.Text != "eu enviei" || got.KeyID != "out1" {
		t.Fatalf("ok=%v got %+v", ok, got)
	}
}

func TestParseInboundFromEvolution_messagesArray(t *testing.T) {
	data := json.RawMessage(`{
		"messages": [{
			"key": {"remoteJid": "a@b", "fromMe": false},
			"message": {"conversation": "hi"}
		}]
	}`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok || got.Text != "hi" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestParseInboundFromEvolution_dataRootArray(t *testing.T) {
	data := json.RawMessage(`[{
		"key": {"remoteJid": "a@b", "fromMe": false},
		"message": {"conversation": "array-root"}
	}]`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok || got.Text != "array-root" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestParseInboundFromEvolution_wrongEvent(t *testing.T) {
	data := json.RawMessage(`{"key":{},"message":{"conversation":"x"}}`)
	_, ok := ParseInboundFromEvolution("connection.update", data)
	if ok {
		t.Fatal("expected false")
	}
}

func TestParseInboundFromEvolution_MESSAGE_event(t *testing.T) {
	data := json.RawMessage(`{
		"key": {"remoteJid": "5511888888888@s.whatsapp.net", "fromMe": false},
		"message": {"conversation": "evogo"}
	}`)
	got, ok := ParseInboundFromEvolution("MESSAGE", data)
	if !ok || got.Text != "evogo" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

// Evolution Go serializa o payload do webhook com PascalCase (Key, Message, RemoteJid, …).
func TestParseInboundFromEvolution_evolutionGoPascalCase(t *testing.T) {
	data := json.RawMessage(`{
		"Key": {"RemoteJid": "5511999999999@s.whatsapp.net", "FromMe": false, "Id": "probe-evogo"},
		"PushName": "Contacto",
		"Message": {"Conversation": "Cuiudu"},
		"MessageTimestamp": 1712966400
	}`)
	got, ok := ParseInboundFromEvolution("Message", data)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.Text != "Cuiudu" || got.From != "5511999999999@s.whatsapp.net" || got.PushName != "Contacto" || got.KeyID != "probe-evogo" {
		t.Fatalf("got %+v", got)
	}
	if got.ReceivedAt.IsZero() {
		t.Fatal("expected MessageTimestamp parsed")
	}
}

// key.id como número JSON (float64) — strFromMap falhava e quebrava getBase64 na Evolution.
func TestParseInboundFromEvolution_keyIdNumeric(t *testing.T) {
	data := json.RawMessage(`{
		"key": {"remoteJid": "5511999999999@s.whatsapp.net", "fromMe": false, "id": 9876543210},
		"message": {"conversation": "ok"}
	}`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok || got.KeyID != "9876543210" || got.Text != "ok" {
		t.Fatalf("ok=%v got %+v", ok, got)
	}
}

// RemoteJid como objeto {user, server} (JSON típico de protobuf / Evolution Go).
func TestParseInboundFromEvolution_remoteJidUserServerObject(t *testing.T) {
	data := json.RawMessage(`{
		"Key": {
			"RemoteJid": {"User": "5569993378283", "Server": "s.whatsapp.net"},
			"FromMe": false,
			"Id": "abc"
		},
		"Message": {"Conversation": "Alo"}
	}`)
	got, ok := ParseInboundFromEvolution("Message", data)
	if !ok {
		t.Fatal("expected ok")
	}
	if want := "5569993378283@s.whatsapp.net"; got.From != want {
		t.Fatalf("From=%q want %q", got.From, want)
	}
	if got.Text != "Alo" {
		t.Fatalf("Text=%q", got.Text)
	}
}

func TestParseInboundFromEvolution_remoteJidInsideMessageMap(t *testing.T) {
	data := json.RawMessage(`{
		"Message": {
			"Key": {"RemoteJid": "5511888888888@s.whatsapp.net", "FromMe": false},
			"Message": {"Conversation": "nested"}
		}
	}`)
	got, ok := ParseInboundFromEvolution("Message", data)
	if !ok || got.Text != "nested" || got.From != "5511888888888@s.whatsapp.net" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestParseInboundFromEvolution_infoRemoteJid(t *testing.T) {
	data := json.RawMessage(`{
		"Info": {"RemoteJid": "5511777777777@s.whatsapp.net"},
		"Message": {"Conversation": "via info"}
	}`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok || got.Text != "via info" || got.From != "5511777777777@s.whatsapp.net" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestParseInboundFromEvolution_ephemeralWrapper(t *testing.T) {
	data := json.RawMessage(`{
		"key": {"remoteJid": "5511999999999@s.whatsapp.net", "fromMe": false},
		"pushName": "Teste",
		"message": {"ephemeralMessage": {"message": {"conversation": "msg oculta"}}}
	}`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok || got.Text != "msg oculta" || got.PushName != "Teste" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestParseInboundFromEvolution_imageCaption(t *testing.T) {
	data := json.RawMessage(`{
		"key": {"remoteJid": "a@s.whatsapp.net", "fromMe": false},
		"message": {"imageMessage": {"caption": "foto com legenda"}}
	}`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok || got.Text != "foto com legenda" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestNormalizeWebhookData_base64StringField(t *testing.T) {
	inner := []byte(`{"key":{"remoteJid":"5511999999999@s.whatsapp.net","fromMe":false},"message":{"conversation":"base64payload"}}`)
	b64 := base64.StdEncoding.EncodeToString(inner)
	quoted, err := json.Marshal(b64)
	if err != nil {
		t.Fatal(err)
	}
	out := NormalizeWebhookData(json.RawMessage(quoted))
	got, ok := ParseInboundFromEvolution("messages.upsert", out)
	if !ok || got.Text != "base64payload" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestNormalizeWebhookData_plainObjectUnchanged(t *testing.T) {
	raw := json.RawMessage(`{"key":{"remoteJid":"x@s.whatsapp.net","fromMe":false},"message":{"conversation":"plain"}}`)
	out := NormalizeWebhookData(raw)
	if string(out) != string(raw) {
		t.Fatalf("expected unchanged, got %s", string(out))
	}
}

func TestParseInboundFromEvolution_remoteJidAlt(t *testing.T) {
	data := json.RawMessage(`{
		"key": {"remoteJid": "111111111111111@lid", "remoteJidAlt": "5569993378283@s.whatsapp.net", "fromMe": false},
		"message": {"conversation": "olá"},
		"messageTimestamp": 1700000000
	}`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok || got.Text != "olá" || got.RemoteJidAlt != "5569993378283@s.whatsapp.net" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
	if got.ReceivedAt != time.Unix(1700000000, 0).UTC() {
		t.Fatalf("timestamp got %v", got.ReceivedAt)
	}
}

func TestParseInboundFromEvolution_nestedDataEnvelope(t *testing.T) {
	data := json.RawMessage(`{
		"data": {
			"key": {"remoteJid": "5511888888888@s.whatsapp.net", "fromMe": true, "id": "nest"},
			"message": {"conversation": "nested env"}
		}
	}`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok || !got.FromMe || got.Text != "nested env" || got.KeyID != "nest" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestNormalizeWebhookData_jsonObjectStringNotBase64(t *testing.T) {
	inner := `{"key":{"remoteJid":"5511888888888@s.whatsapp.net","fromMe":true},"message":{"conversation":"plain json string"}}`
	quoted, err := json.Marshal(inner)
	if err != nil {
		t.Fatal(err)
	}
	out := NormalizeWebhookData(json.RawMessage(quoted))
	got, ok := ParseInboundFromEvolution("messages.upsert", out)
	if !ok || got.Text != "plain json string" || !got.FromMe {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestParseInboundFromEvolution_fromMeImageNoCaption(t *testing.T) {
	data := json.RawMessage(`{
		"key": {"remoteJid": "5511888888888@s.whatsapp.net", "fromMe": true, "id": "img1"},
		"message": {"imageMessage": {"mimetype": "image/jpeg"}}
	}`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok || !got.FromMe || got.Text != "[imagem]" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}

func TestParseInboundFromEvolution_interactiveBody(t *testing.T) {
	data := json.RawMessage(`{
		"key": {"remoteJid": "5511999999999@s.whatsapp.net", "fromMe": true, "id": "int1"},
		"message": {"interactiveMessage": {"body": {"text": "Catálogo"}}}
	}`)
	got, ok := ParseInboundFromEvolution("messages.upsert", data)
	if !ok || !got.FromMe || got.Text != "Catálogo" {
		t.Fatalf("ok=%v got=%+v", ok, got)
	}
}
