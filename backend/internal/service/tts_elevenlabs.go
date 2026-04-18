package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type elevenLabsSpeechBody struct {
	Text          string                   `json:"text"`
	ModelID       string                   `json:"model_id"`
	VoiceSettings *elevenLabsVoiceSettings `json:"voice_settings,omitempty"`
}

type elevenLabsVoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"` // API: similarity_boost
}

// SynthElevenLabs gera áudio via POST https://api.elevenlabs.io/v1/text-to-speech/{voiceID}
// (MP3 por defeito). Documentação: https://elevenlabs.io/docs/api-reference/text-to-speech/convert
func SynthElevenLabs(ctx context.Context, apiKey, voiceID, text, modelID string) ([]byte, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("elevenlabs: api key em falta")
	}
	voiceID = strings.TrimSpace(voiceID)
	if voiceID == "" {
		return nil, fmt.Errorf("elevenlabs: voice_id em falta")
	}
	if modelID == "" {
		modelID = "eleven_multilingual_v2"
	}
	body, err := json.Marshal(elevenLabsSpeechBody{
		Text:    text,
		ModelID: modelID,
		VoiceSettings: &elevenLabsVoiceSettings{
			Stability:       0.5,
			SimilarityBoost: 0.75,
		},
	})
	if err != nil {
		return nil, err
	}
	u := "https://api.elevenlabs.io/v1/text-to-speech/" + voiceID
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Accept", "audio/mpeg")
	hc := &http.Client{Timeout: 120 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 30<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("elevenlabs http %d: %s", resp.StatusCode, truncateErrBody(raw))
	}
	return raw, nil
}
