package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// OpenAIClient implementa LLM via API Chat Completions.
type OpenAIClient struct {
	client *openai.Client
	model  string
	system string
}

func NewOpenAIClient(apiKey, model, systemPrompt string) *OpenAIClient {
	return &OpenAIClient{
		client: openai.NewClient(apiKey),
		model:  model,
		system: systemPrompt,
	}
}

// Reply gera texto de resposta ao utilizador final.
func (o *OpenAIClient) Reply(ctx context.Context, userText string) (string, error) {
	resp, err := o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: o.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: o.system},
			{Role: openai.ChatMessageRoleUser, Content: userText},
		},
		MaxTokens:   512,
		Temperature: 0.4,
	})
	if err != nil {
		return "", fmt.Errorf("openai: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai: resposta vazia")
	}
	return resp.Choices[0].Message.Content, nil
}

// TranscribeAudio usa o modelo whisper-1.
func (o *OpenAIClient) TranscribeAudio(ctx context.Context, audio []byte, mimeType string) (string, error) {
	if len(audio) == 0 {
		return "", fmt.Errorf("áudio vazio")
	}
	ext := ".ogg"
	if m := strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0])); m != "" {
		switch m {
		case "audio/mpeg", "audio/mp3":
			ext = ".mp3"
		case "audio/mp4", "audio/aac", "audio/m4a":
			ext = ".m4a"
		case "audio/wav", "audio/x-wav":
			ext = ".wav"
		case "audio/webm":
			ext = ".webm"
		}
	}
	resp, err := o.client.CreateTranscription(ctx, openai.AudioRequest{
		Model:    openai.Whisper1,
		Reader:   bytes.NewReader(audio),
		FilePath: "voice" + ext,
		Format:   openai.AudioResponseFormatText,
	})
	if err != nil {
		return "", fmt.Errorf("openai whisper: %w", err)
	}
	out := strings.TrimSpace(resp.Text)
	if out == "" {
		return "", fmt.Errorf("openai whisper: transcrição vazia")
	}
	return out, nil
}
