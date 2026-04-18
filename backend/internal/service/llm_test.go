package service

import (
	"testing"

	"wa-saas/backend/internal/config"
)

func TestNewLLM_autoReplyOff(t *testing.T) {
	llm, err := NewLLM(&config.Config{AutoReplyEnabled: false})
	if err != nil || llm != nil {
		t.Fatalf("err=%v llm=%v", err, llm)
	}
}

func TestNewLLM_autoReplyOn_geminiNoEnvKey(t *testing.T) {
	llm, err := NewLLM(&config.Config{
		AutoReplyEnabled: true,
		LLMProvider:      "gemini",
		GeminiAPIKey:     "",
		GeminiModel:      "gemini-2.0-flash",
		LLMSystemPrompt:  "x",
	})
	if err != nil || llm != nil {
		t.Fatalf("err=%v llm=%v (esperado nil LLM; chave pode vir do agente)", err, llm)
	}
}

func TestProviderName(t *testing.T) {
	if ProviderName(NewGeminiClient("k", "m", "s")) != "gemini" {
		t.Fatal()
	}
	if ProviderName(NewOpenAIClient("k", "m", "s")) != "openai" {
		t.Fatal()
	}
	if ProviderName(nil) != "unknown" {
		t.Fatal()
	}
}
