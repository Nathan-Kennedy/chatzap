package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseEvolutionBase64MediaResponse_plain(t *testing.T) {
	payload := map[string]string{"base64": base64.StdEncoding.EncodeToString([]byte{1, 2, 3})}
	raw, _ := json.Marshal(payload)
	dec, mime, err := ParseEvolutionBase64MediaResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if mime != "" {
		t.Fatalf("mime=%q", mime)
	}
	if string(dec) != string([]byte{1, 2, 3}) {
		t.Fatalf("got %v", dec)
	}
}

func TestParseEvolutionBase64MediaResponse_dataURL(t *testing.T) {
	b := base64.StdEncoding.EncodeToString([]byte("hello"))
	raw := []byte(`{"base64":"data:image/jpeg;base64,` + b + `"}`)
	dec, mime, err := ParseEvolutionBase64MediaResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if mime != "image/jpeg" {
		t.Fatalf("mime=%q", mime)
	}
	if string(dec) != "hello" {
		t.Fatalf("got %q", dec)
	}
}

func TestParseEvolutionBase64MediaResponse_nestedMessage(t *testing.T) {
	b := base64.StdEncoding.EncodeToString([]byte("nested"))
	raw := []byte(`{"message":{"base64":"` + b + `","mimetype":"audio/ogg"}}`)
	dec, mime, err := ParseEvolutionBase64MediaResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if mime != "audio/ogg" {
		t.Fatalf("mime=%q", mime)
	}
	if string(dec) != "nested" {
		t.Fatalf("got %q", dec)
	}
}

// Envelope típico do Evolution Go em POST /message/downloadmedia.
func TestParseEvolutionBase64MediaResponse_evolutionGoDataEnvelope(t *testing.T) {
	inner := base64.StdEncoding.EncodeToString([]byte{9, 9, 9})
	raw := []byte(`{"message":"success","data":{"base64":"data:audio/ogg;base64,` + inner + `"}}`)
	dec, mime, err := ParseEvolutionBase64MediaResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(mime), "ogg") {
		t.Fatalf("mime=%q", mime)
	}
	if len(dec) != 3 || dec[0] != 9 {
		t.Fatalf("got %v", dec)
	}
}
