package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiClient_Reply(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "test-key" {
			t.Fatal("missing key")
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{{"text": "Olá do mock"}},
					},
				},
			},
		})
	}))
	defer ts.Close()

	g := &GeminiClient{
		apiKey:     "test-key",
		model:      "gemini-2.0-flash",
		system:     "Seja breve.",
		httpClient: ts.Client(),
		baseURL:    ts.URL,
	}
	out, err := g.Reply(context.Background(), "Oi")
	if err != nil {
		t.Fatal(err)
	}
	if out != "Olá do mock" {
		t.Fatalf("got %q", out)
	}
}

func TestGeminiClient_Reply_retriesHighDemand(t *testing.T) {
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    503,
					"message": "This model is currently experiencing high demand. Please try again later.",
					"status":  "UNAVAILABLE",
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates": []map[string]interface{}{
				{"content": map[string]interface{}{"parts": []map[string]string{{"text": "ok"}}}},
			},
		})
	}))
	defer ts.Close()

	g := &GeminiClient{
		apiKey:     "k",
		model:      "m",
		system:     "",
		httpClient: ts.Client(),
		baseURL:    ts.URL,
	}
	out, err := g.Reply(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok" {
		t.Fatalf("got %q calls=%d", out, n)
	}
	if n != 2 {
		t.Fatalf("esperava 2 pedidos HTTP, got %d", n)
	}
}

func TestGeminiErrorRetriable(t *testing.T) {
	if !geminiErrorRetriable(fmt.Errorf("gemini api: high demand please wait")) {
		t.Fatal()
	}
	if geminiErrorRetriable(fmt.Errorf("gemini api: invalid api key")) {
		t.Fatal()
	}
}
