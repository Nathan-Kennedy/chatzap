package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	geminiDefaultBase  = "https://generativelanguage.googleapis.com/v1beta"
	geminiMaxRetries   = 4
	geminiRetryMaxWait = 12 * time.Second
)

// GeminiClient chama a API generateContent (Google AI / Gemini).
type GeminiClient struct {
	apiKey     string
	model      string
	system     string
	httpClient *http.Client
	baseURL    string
}

func NewGeminiClient(apiKey, model, systemPrompt string) *GeminiClient {
	if strings.TrimSpace(model) == "" {
		model = "gemini-2.5-flash"
	}
	return &GeminiClient{
		apiKey: strings.TrimSpace(apiKey),
		model:  model,
		system: systemPrompt,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
		baseURL: geminiDefaultBase,
	}
}

type geminiGenerateRequest struct {
	SystemInstruction *geminiContent `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  struct {
		Temperature     float32 `json:"temperature"`
		MaxOutputTokens int     `json:"maxOutputTokens"`
	} `json:"generationConfig"`
}

type geminiContent struct {
	Role  string        `json:"role,omitempty"`
	Parts []geminiPart  `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// Reply implementa LLM. Reintenta em erros transitórios (pico de procura, 429, 503).
func (g *GeminiClient) Reply(ctx context.Context, userText string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < geminiMaxRetries; attempt++ {
		if attempt > 0 {
			// 1s, 2s, 4s, … até geminiRetryMaxWait
			wait := time.Duration(1<<uint(attempt-1)) * time.Second
			if wait > geminiRetryMaxWait {
				wait = geminiRetryMaxWait
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(wait):
			}
		}
		s, err := g.generateOnce(ctx, userText)
		if err == nil {
			return s, nil
		}
		lastErr = err
		if !geminiErrorRetriable(err) {
			return "", err
		}
	}
	return "", lastErr
}

func (g *GeminiClient) generateOnce(ctx context.Context, userText string) (string, error) {
	base := strings.TrimSuffix(strings.TrimSpace(g.baseURL), "/")
	endpoint := fmt.Sprintf("%s/models/%s:generateContent", base, g.model)
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("key", g.apiKey)
	u.RawQuery = q.Encode()

	var body geminiGenerateRequest
	if strings.TrimSpace(g.system) != "" {
		body.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: g.system}},
		}
	}
	body.Contents = []geminiContent{
		{Role: "user", Parts: []geminiPart{{Text: userText}}},
	}
	body.GenerationConfig.Temperature = 0.4
	body.GenerationConfig.MaxOutputTokens = 512

	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini http: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<22))
	if err != nil {
		return "", err
	}

	var parsed geminiGenerateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("gemini json: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("gemini api: %s", parsed.Error.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini status %d: %s", resp.StatusCode, string(respBody))
	}
	if len(parsed.Candidates) == 0 {
		return "", fmt.Errorf("gemini: resposta sem candidates")
	}
	var out strings.Builder
	for _, p := range parsed.Candidates[0].Content.Parts {
		out.WriteString(p.Text)
	}
	s := strings.TrimSpace(out.String())
	if s == "" {
		return "", fmt.Errorf("gemini: texto vazio")
	}
	return s, nil
}

// TranscribeAudio envia o áudio como generateContent (tenta camelCase REST, depois snake_case).
func (g *GeminiClient) TranscribeAudio(ctx context.Context, audio []byte, mimeType string) (string, error) {
	if len(audio) == 0 {
		return "", fmt.Errorf("áudio vazio")
	}
	mime := strings.TrimSpace(strings.Split(mimeType, ";")[0])
	if mime == "" {
		mime = "audio/ogg"
	}
	b64 := base64.StdEncoding.EncodeToString(audio)
	prompt := "Transcreva integralmente o áudio. Responda apenas com o texto falado (português se for o caso), sem comentários nem prefixos."
	partVariants := [][]interface{}{
		{
			map[string]interface{}{
				"inlineData": map[string]string{"mimeType": mime, "data": b64},
			},
			map[string]string{"text": prompt},
		},
		{
			map[string]interface{}{
				"inline_data": map[string]string{"mime_type": mime, "data": b64},
			},
			map[string]string{"text": prompt},
		},
	}
	var lastErr error
	for _, parts := range partVariants {
		s, err := g.transcribeAudioParts(ctx, parts)
		if err == nil && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s), nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("gemini transcribe: texto vazio")
}

func (g *GeminiClient) transcribeAudioParts(ctx context.Context, parts []interface{}) (string, error) {
	base := strings.TrimSuffix(strings.TrimSpace(g.baseURL), "/")
	endpoint := fmt.Sprintf("%s/models/%s:generateContent", base, g.model)
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("key", g.apiKey)
	u.RawQuery = q.Encode()
	content := map[string]interface{}{
		"role":  "user",
		"parts": parts,
	}
	body := map[string]interface{}{
		"contents": []interface{}{content},
		"generationConfig": map[string]interface{}{
			"temperature":     0.2,
			"maxOutputTokens": 2048,
		},
	}
	if strings.TrimSpace(g.system) != "" {
		body["systemInstruction"] = map[string]interface{}{
			"parts": []interface{}{map[string]string{"text": g.system}},
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini transcribe http: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<22))
	if err != nil {
		return "", err
	}
	var parsed geminiGenerateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("gemini transcribe json: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("gemini transcribe api: %s", parsed.Error.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini transcribe status %d: %s", resp.StatusCode, string(respBody))
	}
	if len(parsed.Candidates) == 0 {
		return "", fmt.Errorf("gemini transcribe: sem candidates")
	}
	var out strings.Builder
	for _, p := range parsed.Candidates[0].Content.Parts {
		out.WriteString(p.Text)
	}
	s := strings.TrimSpace(out.String())
	if s == "" {
		return "", fmt.Errorf("gemini transcribe: texto vazio")
	}
	return s, nil
}

func geminiErrorRetriable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "high demand"),
		strings.Contains(msg, "try again later"),
		strings.Contains(msg, "try again in"),
		strings.Contains(msg, "resource_exhausted"),
		strings.Contains(msg, "resource exhausted"),
		strings.Contains(msg, "unavailable"),
		strings.Contains(msg, "overloaded"),
		strings.Contains(msg, "deadline exceeded"),
		strings.Contains(msg, "status 429"),
		strings.Contains(msg, "status 503"),
		strings.Contains(msg, "status 529"):
		return true
	default:
		return false
	}
}
