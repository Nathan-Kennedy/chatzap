package service

import (
	"context"
	"fmt"
	"strings"

	"wa-saas/backend/internal/config"
)

// LLM gera texto de resposta para o utilizador final (Gemini ou OpenAI).
type LLM interface {
	Reply(ctx context.Context, userText string) (string, error)
}

// ProviderName identifica o backend LLM (logs / meta).
func ProviderName(llm LLM) string {
	switch llm.(type) {
	case *GeminiClient:
		return "gemini"
	case *OpenAIClient:
		return "openai"
	default:
		return "unknown"
	}
}

// NewLLM instancia o cliente conforme cfg.LLMProvider (gemini por defeito).
func NewLLM(cfg *config.Config) (LLM, error) {
	if !cfg.AutoReplyEnabled {
		return nil, nil
	}
	switch cfg.LLMProvider {
	case "gemini":
		if strings.TrimSpace(cfg.GeminiAPIKey) == "" {
			return nil, nil
		}
		return NewGeminiClient(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.LLMSystemPrompt), nil
	case "openai":
		if strings.TrimSpace(cfg.OpenAIAPIKey) == "" {
			return nil, nil
		}
		return NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.LLMSystemPrompt), nil
	default:
		return nil, fmt.Errorf("LLM_PROVIDER inválido: %q (use gemini ou openai)", cfg.LLMProvider)
	}
}
