package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseFindMessagesResponse_array(t *testing.T) {
	raw := json.RawMessage(`[{
		"key": {"remoteJid": "5569993378283@s.whatsapp.net", "fromMe": false, "id": "msg1"},
		"message": {"conversation": "oi"},
		"messageTimestamp": 1700000000
	}]`)
	items, err := ParseFindMessagesResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Body != "oi" || items[0].Direction != "inbound" || items[0].ExternalID != "msg1" {
		t.Fatalf("got %+v", items)
	}
}

func TestParseFindMessagesResponse_wrapped(t *testing.T) {
	raw := json.RawMessage(`{"data":{"messages":[{"key":{"fromMe":true,"id":"x"},"message":{"conversation":"saida"}}]}}`)
	items, err := ParseFindMessagesResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Direction != "outbound" || items[0].Body != "saida" {
		t.Fatalf("got %+v", items)
	}
}

func TestParseFindMessagesResponse_pascalCaseKey(t *testing.T) {
	raw := json.RawMessage(`[{"Key":{"FromMe":false,"Id":"abc2"},"Message":{"Conversation":"ola"},"MessageTimestamp":1700000001}]`)
	items, err := ParseFindMessagesResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ExternalID != "abc2" || items[0].Body != "ola" {
		t.Fatalf("got %+v", items)
	}
}

func TestParseFindMessagesResponse_imageMessage_mediaURL(t *testing.T) {
	raw := json.RawMessage(`[{
		"key": {"remoteJid": "5569993378283@s.whatsapp.net", "fromMe": false, "id": "img1"},
		"message": {
			"imageMessage": {
				"url": "https://mmg.whatsapp.net/v/t62.7118-24/xxx",
				"mimetype": "image/jpeg",
				"caption": "foto do dia"
			}
		},
		"messageTimestamp": 1700000002
	}]`)
	items, err := ParseFindMessagesResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d %+v", len(items), items)
	}
	it := items[0]
	if it.MessageType != "image" {
		t.Fatalf("MessageType=%q", it.MessageType)
	}
	if it.Body != "foto do dia" {
		t.Fatalf("Body=%q want caption", it.Body)
	}
	if it.MediaRemoteURL == "" || !strings.Contains(it.MediaRemoteURL, "mmg.whatsapp.net") {
		t.Fatalf("MediaRemoteURL=%q", it.MediaRemoteURL)
	}
}
