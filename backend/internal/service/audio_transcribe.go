package service

import (
	"context"
	"fmt"
	"strings"

	"wa-saas/backend/internal/config"
)

// TranscribeVoiceNote converte áudio em texto (Gemini multimodal ou OpenAI Whisper).
func TranscribeVoiceNote(ctx context.Context, llm LLM, audio []byte, mimeType string) (string, error) {
	if len(audio) == 0 {
		return "", fmt.Errorf("áudio vazio")
	}
	switch v := llm.(type) {
	case *GeminiClient:
		return v.TranscribeAudio(ctx, audio, mimeType)
	case *OpenAIClient:
		return v.TranscribeAudio(ctx, audio, mimeType)
	default:
		return "", fmt.Errorf("transcrição de voz não suportada para este fornecedor de LLM")
	}
}

// TranscribeVoiceNoteWithConfig tenta o LLM configurado (ex. agente) e, se falhar ou for vazio,
// usa GEMINI_API_KEY + GEMINI_MODEL globais — útil quando o agente de auto-reply não é Gemini.
func TranscribeVoiceNoteWithConfig(ctx context.Context, llm LLM, cfg *config.Config, audio []byte, mimeType string) (string, error) {
	if len(audio) == 0 {
		return "", fmt.Errorf("áudio vazio")
	}
	if cfg == nil {
		return TranscribeVoiceNote(ctx, llm, audio, mimeType)
	}
	s, err := TranscribeVoiceNote(ctx, llm, audio, mimeType)
	if err == nil {
		if t := strings.TrimSpace(s); t != "" {
			return t, nil
		}
	}
	firstErr := err
	key := strings.TrimSpace(cfg.GeminiAPIKey)
	if key == "" {
		if firstErr != nil {
			return "", firstErr
		}
		return "", fmt.Errorf("transcrição vazia e sem GEMINI_API_KEY no .env")
	}
	g := NewGeminiClient(key, cfg.GeminiModel, "")
	return g.TranscribeAudio(ctx, audio, mimeType)
}
