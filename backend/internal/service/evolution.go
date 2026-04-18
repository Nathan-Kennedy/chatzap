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

// EvolutionClient chama a Evolution API v2 (sendText).
type EvolutionClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewEvolutionClient(baseURL, apiKey string) *EvolutionClient {
	return &EvolutionClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

type sendTextRequest struct {
	Number      string `json:"number"`
	Text        string `json:"text"`
	LinkPreview *bool  `json:"linkPreview,omitempty"`
}

// Formato plano (Evolution API Node / validação presenceSchema).
type sendPresenceRequestFlat struct {
	Number   string `json:"number"`
	Presence string `json:"presence"`
	Delay    int    `json:"delay"`
}

// Formato documentado Evo Cloud / Evolution Go: https://docs.evoapicloud.com/api-reference/chat-controller/send-presence
type sendPresenceRequestNested struct {
	Number  string `json:"number"`
	Options struct {
		Delay    int    `json:"delay"`
		Presence string `json:"presence"`
		Number   string `json:"number"`
	} `json:"options"`
}

// chatPresenceGoRequest Evolution Go (whatsmeow): POST /message/presence — ChatPresenceStruct.
type chatPresenceGoRequest struct {
	Number  string `json:"number"`
	State   string `json:"state"`
	IsAudio bool   `json:"isAudio"`
}

// SendPresence envia estado de digitação/gravação.
// 1) Evolution Go: POST /message/presence (state + isAudio).
// 2) Se 404 (rota inexistente), fallback Node: POST /chat/sendPresence/{instanceName}.
// delayMs: só usado no formato Node (options/delay). presence: composing | recording | paused | available.
func (c *EvolutionClient) SendPresence(ctx context.Context, instanceName, instanceToken, jidOrNumber, presence string, delayMs int) error {
	name := strings.TrimSpace(instanceName)
	if name == "" {
		return fmt.Errorf("instância vazia")
	}
	number := normalizeWhatsAppNumber(jidOrNumber)
	if delayMs < 1 {
		delayMs = 1
	}
	p := strings.TrimSpace(presence)
	if p == "" {
		p = "composing"
	}
	token := strings.TrimSpace(instanceToken)
	if token == "" {
		token = c.apiKey
	}

	uGo := fmt.Sprintf("%s/message/presence", c.baseURL)
	rawGo, err := json.Marshal(chatPresenceGoRequest{
		Number:  number,
		State:   p,
		IsAudio: strings.EqualFold(p, "recording"),
	})
	if err == nil {
		if errGo := c.postSendPresence(ctx, uGo, token, rawGo); errGo == nil {
			return nil
		} else if !sendPresenceErrIs404(errGo) {
			return errGo
		}
	}

	u := fmt.Sprintf("%s/chat/sendPresence/%s", c.baseURL, url.PathEscape(name))

	var nested sendPresenceRequestNested
	nested.Number = number
	nested.Options.Delay = delayMs
	nested.Options.Presence = p
	nested.Options.Number = number
	rawNested, err := json.Marshal(nested)
	if err != nil {
		return err
	}
	errNested := c.postSendPresence(ctx, u, token, rawNested)
	if errNested == nil {
		return nil
	}
	if !sendPresenceTryFlatBody(errNested) {
		return errNested
	}
	rawFlat, err := json.Marshal(sendPresenceRequestFlat{
		Number:   number,
		Presence: p,
		Delay:    delayMs,
	})
	if err != nil {
		return errNested
	}
	errFlat := c.postSendPresence(ctx, u, token, rawFlat)
	if errFlat == nil {
		return nil
	}
	return fmt.Errorf("%v; corpo plano: %w", errNested, errFlat)
}

// Só faz fallback ao JSON plano quando o erro parece validação de body (não 404).
func sendPresenceTryFlatBody(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	for _, code := range []string{"400", "422", "415"} {
		if strings.Contains(s, "status "+code) {
			return true
		}
	}
	return false
}

func sendPresenceErrIs404(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "status 404")
}

func (c *EvolutionClient) postSendPresence(ctx context.Context, urlStr, apiToken string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", apiToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("evolution sendPresence http: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		trim := raw
		if len(trim) > 500 {
			trim = trim[:500]
		}
		return fmt.Errorf("evolution sendPresence status %d: %s", resp.StatusCode, string(trim))
	}
	return nil
}

// SendText POST /send/text com header apikey (token da instância no Evolution Go).
// Devolve o corpo JSON da resposta (para key.id / remoteJid) ou nil se vazio.
func (c *EvolutionClient) SendText(ctx context.Context, instanceToken, number, text string) ([]byte, error) {
	number = normalizeWhatsAppNumber(number)
	u := fmt.Sprintf("%s/send/text", c.baseURL)
	body, err := json.Marshal(sendTextRequest{Number: number, Text: text})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	token := strings.TrimSpace(instanceToken)
	if token == "" {
		token = c.apiKey
	}
	req.Header.Set("apikey", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("evolution http: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return raw, fmt.Errorf("evolution status %d: %s", resp.StatusCode, string(raw))
	}
	return raw, nil
}

type sendMediaRequest struct {
	Number   string `json:"number"`
	Type     string `json:"type"` // image | video | audio | document
	URL      string `json:"url"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// SendMedia POST /send/media (Evolution Go).
func (c *EvolutionClient) SendMedia(ctx context.Context, instanceToken, number, mediaType, fileURL, caption, filename string) ([]byte, error) {
	number = normalizeWhatsAppNumber(number)
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	u := fmt.Sprintf("%s/send/media", c.baseURL)
	body, err := json.Marshal(sendMediaRequest{
		Number:   number,
		Type:     mediaType,
		URL:      strings.TrimSpace(fileURL),
		Caption:  strings.TrimSpace(caption),
		Filename: strings.TrimSpace(filename),
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	token := strings.TrimSpace(instanceToken)
	if token == "" {
		token = c.apiKey
	}
	req.Header.Set("apikey", token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("evolution http: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return raw, fmt.Errorf("evolution status %d: %s", resp.StatusCode, string(raw))
	}
	return raw, nil
}

// evolutionMediaPayloadMinimal corresponde ao OpenAPI v2 (só message.key.id + convertToMp4).
func evolutionMediaPayloadMinimal(messageKeyID string, convertToMp4 bool) map[string]interface{} {
	return map[string]interface{}{
		"message": map[string]interface{}{
			"key": map[string]interface{}{
				"id": strings.TrimSpace(messageKeyID),
			},
		},
		"convertToMp4": convertToMp4,
	}
}

// evolutionMediaPayloadExtended inclui remoteJid/fromMe (compat Baileys quando o minimal falha).
func evolutionMediaPayloadExtended(messageKeyID, remoteJid string, fromMe, convertToMp4 bool) map[string]interface{} {
	key := map[string]interface{}{
		"id": strings.TrimSpace(messageKeyID),
	}
	if rj := strings.TrimSpace(remoteJid); rj != "" {
		key["remoteJid"] = rj
	}
	key["fromMe"] = fromMe
	return map[string]interface{}{
		"message": map[string]interface{}{
			"key": key,
		},
		"convertToMp4": convertToMp4,
	}
}

func (c *EvolutionClient) postGetBase64FromMediaMessage(ctx context.Context, instanceName, instanceToken string, payload map[string]interface{}) (raw []byte, status int, err error) {
	name := strings.TrimSpace(instanceName)
	rawBody, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	u := fmt.Sprintf("%s/chat/getBase64FromMediaMessage/%s", c.baseURL, url.PathEscape(name))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(rawBody))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	token := strings.TrimSpace(instanceToken)
	if token == "" {
		token = c.apiKey
	}
	req.Header.Set("apikey", token)
	hc := &http.Client{Timeout: 120 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("evolution http: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 30<<20))
	return body, resp.StatusCode, nil
}

// GetBase64FromMediaMessage POST /chat/getBase64FromMediaMessage/{instance} — obtém bytes da mídia (mensagens recebidas onde a URL mmg.whatsapp.net não é acessível sem o Baileys).
// Tenta primeiro o corpo mínimo da documentação Evolution v2; se 4xx/5xx ou base64 inválido, tenta com remoteJid/fromMe.
func (c *EvolutionClient) GetBase64FromMediaMessage(ctx context.Context, instanceName, instanceToken, messageKeyID, remoteJid string, fromMe, convertToMp4 bool) (media []byte, mimeHint string, err error) {
	name := strings.TrimSpace(instanceName)
	kid := strings.TrimSpace(messageKeyID)
	if name == "" || kid == "" {
		return nil, "", fmt.Errorf("instância ou key.id vazio")
	}
	payloads := []map[string]interface{}{
		evolutionMediaPayloadMinimal(kid, convertToMp4),
		evolutionMediaPayloadExtended(kid, remoteJid, fromMe, convertToMp4),
	}
	var lastErr error
	for i, payload := range payloads {
		raw, status, reqErr := c.postGetBase64FromMediaMessage(ctx, name, instanceToken, payload)
		if reqErr != nil {
			lastErr = reqErr
			if i == 0 {
				continue
			}
			return nil, "", lastErr
		}
		if status < 200 || status >= 300 {
			trim := bytes.TrimSpace(raw)
			if len(trim) > 400 {
				trim = trim[:400]
			}
			lastErr = fmt.Errorf("evolution getBase64 status %d: %s", status, string(trim))
			if i == 0 {
				continue
			}
			return nil, "", lastErr
		}
		decoded, mime, perr := ParseEvolutionBase64MediaResponse(raw)
		if perr == nil && len(decoded) > 0 {
			return decoded, mime, nil
		}
		if perr != nil {
			lastErr = perr
		} else {
			lastErr = fmt.Errorf("resposta vazia")
		}
		if i == 0 {
			continue
		}
		return nil, "", lastErr
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("evolution getBase64: sem resposta")
	}
	return nil, "", lastErr
}

// DownloadMediaEvolutionGo POST /message/downloadmedia — Evolution Go (whatsmeow).
// O corpo é {"message": <JSON do proto>}; o apikey deve ser o token da instância.
func (c *EvolutionClient) DownloadMediaEvolutionGo(ctx context.Context, instanceToken string, waMessageJSON []byte) ([]byte, string, error) {
	waMessageJSON = bytes.TrimSpace(waMessageJSON)
	if len(waMessageJSON) == 0 {
		return nil, "", fmt.Errorf("mensagem whatsapp json vazia")
	}
	if !json.Valid(waMessageJSON) {
		return nil, "", fmt.Errorf("mensagem whatsapp json inválida")
	}
	inner := json.RawMessage(append([]byte(nil), waMessageJSON...))
	wrapped, err := json.Marshal(map[string]json.RawMessage{"message": inner})
	if err != nil {
		return nil, "", err
	}
	u := fmt.Sprintf("%s/message/downloadmedia", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(wrapped))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	token := strings.TrimSpace(instanceToken)
	if token == "" {
		token = c.apiKey
	}
	req.Header.Set("apikey", token)
	hc := &http.Client{Timeout: 120 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("evolution downloadmedia http: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 30<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		trim := bytes.TrimSpace(raw)
		if len(trim) > 400 {
			trim = trim[:400]
		}
		return nil, "", fmt.Errorf("evolution downloadmedia status %d: %s", resp.StatusCode, string(trim))
	}
	return ParseEvolutionBase64MediaResponse(raw)
}

// ParseEvolutionBase64MediaResponse interpreta JSON típico da Evolution com campo base64 (ou data URL).
func ParseEvolutionBase64MediaResponse(raw []byte) (decoded []byte, mime string, err error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, "", fmt.Errorf("resposta vazia")
	}
	var root map[string]interface{}
	if json.Unmarshal(raw, &root) != nil {
		return nil, "", fmt.Errorf("json inválido")
	}
	b64 := strFromMap(root, "base64", "Base64")
	if b64 == "" {
		if d, ok := root["data"].(map[string]interface{}); ok {
			b64 = strFromMap(d, "base64", "Base64")
		}
	}
	mime = strFromMap(root, "mimetype", "Mimetype", "mimeType", "MimeType")
	if mime == "" {
		if d, ok := root["data"].(map[string]interface{}); ok {
			mime = strFromMap(d, "mimetype", "Mimetype", "mimeType", "MimeType")
		}
	}
	if b64 == "" {
		if msg, ok := root["message"].(map[string]interface{}); ok {
			b64 = strFromMap(msg, "base64", "Base64")
			if mime == "" {
				mime = strFromMap(msg, "mimetype", "Mimetype", "mimeType", "MimeType")
			}
		}
	}
	if b64 == "" {
		if r, ok := root["result"].(map[string]interface{}); ok {
			b64 = strFromMap(r, "base64", "Base64")
			if mime == "" {
				mime = strFromMap(r, "mimetype", "Mimetype", "mimeType", "MimeType")
			}
		}
	}
	if b64 == "" {
		return nil, "", fmt.Errorf("campo base64 em falta na resposta")
	}
	if i := strings.Index(b64, ";base64,"); i >= 0 && strings.HasPrefix(b64, "data:") {
		prefix := strings.TrimPrefix(b64[:i], "data:")
		if mime == "" && prefix != "" {
			mime = prefix
		}
		b64 = b64[i+len(";base64,"):]
	}
	b64 = strings.TrimSpace(b64)
	decoded, err = base64.StdEncoding.DecodeString(b64)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(b64)
	}
	if err != nil {
		return nil, "", fmt.Errorf("base64: %w", err)
	}
	return decoded, strings.TrimSpace(mime), nil
}

// ParseEvolutionSendTextResponse extrai JID e id da mensagem de respostas típicas da Evolution / Baileys.
func ParseEvolutionSendTextResponse(raw []byte) (remoteJid, keyID string) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return "", ""
	}
	var root map[string]interface{}
	if json.Unmarshal(raw, &root) != nil {
		return "", ""
	}
	m := root
	if d, ok := root["data"].(map[string]interface{}); ok {
		m = d
	}
	key := mapByKeys(m, "key", "Key")
	if key == nil {
		return "", ""
	}
	rj := jidFromFlexible(key, "remoteJid", "RemoteJid", "remote_jid")
	kid := keyIDFromKeyMap(key)
	if kid == "" {
		if s, ok := root["messageId"].(string); ok && strings.TrimSpace(s) != "" {
			kid = strings.TrimSpace(s)
		}
	}
	if kid == "" && m != nil {
		if info, ok := m["Info"].(map[string]interface{}); ok {
			kid = keyIDFromKeyMap(info)
		}
	}
	return rj, kid
}

func normalizeWhatsAppNumber(jidOrNumber string) string {
	s := strings.TrimSpace(jidOrNumber)
	s = strings.Split(s, "@")[0]
	s = strings.TrimPrefix(s, "+")
	return s
}
