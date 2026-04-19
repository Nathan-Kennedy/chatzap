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
	"unicode"
)

// ElevenLabsTelephonyCallConfig telephony_call_config na API ConvAI.
type ElevenLabsTelephonyCallConfig struct {
	RingingTimeoutSecs int `json:"ringing_timeout_secs"`
}

// ElevenLabsTwilioOutboundCallRequest corpo POST /v1/convai/twilio/outbound-call.
type ElevenLabsTwilioOutboundCallRequest struct {
	AgentID             string                         `json:"agent_id"`
	AgentPhoneNumberID  string                         `json:"agent_phone_number_id"`
	ToNumber            string                         `json:"to_number"`
	CallRecording       *bool                          `json:"call_recording_enabled,omitempty"`
	TelephonyCallConfig *ElevenLabsTelephonyCallConfig `json:"telephony_call_config,omitempty"`
}

// ElevenLabsTwilioOutboundCallResponse resposta JSON da ElevenLabs.
type ElevenLabsTwilioOutboundCallResponse struct {
	Success        bool    `json:"success"`
	Message        string  `json:"message"`
	ConversationID *string `json:"conversation_id"`
	CallSid        *string `json:"callSid"`
}

// NormalizeToE164 normaliza para E.164 (+ e apenas dígitos após o +).
func NormalizeToE164(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("número vazio")
	}
	var b strings.Builder
	if strings.HasPrefix(s, "+") {
		b.WriteByte('+')
		s = s[1:]
	}
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if !strings.HasPrefix(out, "+") {
		out = "+" + out
	}
	digits := len(out) - 1
	if digits < 8 || digits > 15 {
		return "", fmt.Errorf("número internacional inválido (esperado 8–15 dígitos após o +)")
	}
	return out, nil
}

// ElevenLabsTwilioOutboundCall inicia uma chamada telefónica outbound via Twilio (ConvAI).
// apiBaseURL ex.: https://api.elevenlabs.io (ou endpoint de residência de dados).
func ElevenLabsTwilioOutboundCall(
	ctx context.Context,
	apiKey string,
	apiBaseURL string,
	req ElevenLabsTwilioOutboundCallRequest,
) (*ElevenLabsTwilioOutboundCallResponse, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("elevenlabs: api key em falta")
	}
	apiBaseURL = strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")
	if apiBaseURL == "" {
		apiBaseURL = "https://api.elevenlabs.io"
	}
	to, err := NormalizeToE164(req.ToNumber)
	if err != nil {
		return nil, err
	}
	req.ToNumber = to
	if strings.TrimSpace(req.AgentID) == "" || strings.TrimSpace(req.AgentPhoneNumberID) == "" {
		return nil, fmt.Errorf("elevenlabs convai: agent_id e agent_phone_number_id são obrigatórios")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	u := apiBaseURL + "/v1/convai/twilio/outbound-call"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("xi-api-key", apiKey)

	hc := &http.Client{Timeout: 60 * time.Second}
	resp, err := hc.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("elevenlabs convai outbound http %d: %s", resp.StatusCode, truncateErrBody(raw))
	}
	var out ElevenLabsTwilioOutboundCallResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("elevenlabs convai: resposta inválida: %w", err)
	}
	if !out.Success {
		msg := strings.TrimSpace(out.Message)
		if msg == "" {
			msg = "success=false sem mensagem"
		}
		return nil, fmt.Errorf("elevenlabs convai: %s", msg)
	}
	return &out, nil
}
