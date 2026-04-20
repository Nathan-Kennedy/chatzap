package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EvolutionInstanceInfo resumo devolvido pela Evolution Go (subset).
type EvolutionInstanceInfo struct {
	Name       string `json:"name"`
	InstanceID string `json:"id"`
	Status     string `json:"status"`
	OwnerJID   string `json:"jid"`
	Connected  bool   `json:"connected"`
}

// CreateInstanceRequest body POST /instance/create (Evolution Go).
type CreateInstanceRequest struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

// CreateInstance POST /instance/create com apikey global.
func (c *EvolutionClient) CreateInstance(ctx context.Context, instanceName, instanceToken string) error {
	body, err := json.Marshal(CreateInstanceRequest{
		Name:  instanceName,
		Token: instanceToken,
	})
	if err != nil {
		return err
	}
	u := fmt.Sprintf("%s/instance/create", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("evolution create instance %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

type dataWrap struct {
	Data json.RawMessage `json:"data"`
}

// FetchInstances GET /instance/all (lista JSON).
func (c *EvolutionClient) FetchInstances(ctx context.Context) ([]EvolutionInstanceInfo, error) {
	u := fmt.Sprintf("%s/instance/all", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("evolution list instances %d: %s", resp.StatusCode, string(raw))
	}
	var out []EvolutionInstanceInfo
	if err := json.Unmarshal(raw, &out); err == nil {
		return out, nil
	}
	var wrap dataWrap
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Data) > 0 {
		if err := json.Unmarshal(wrap.Data, &out); err == nil {
			return out, nil
		}
		var one EvolutionInstanceInfo
		if err := json.Unmarshal(wrap.Data, &one); err == nil && one.Name != "" {
			return []EvolutionInstanceInfo{one}, nil
		}
	}
	var one EvolutionInstanceInfo
	if err := json.Unmarshal(raw, &one); err == nil && one.Name != "" {
		return []EvolutionInstanceInfo{one}, nil
	}
	return nil, fmt.Errorf("decode list instances")
}

type qrData struct {
	Code         string `json:"code"`
	Qrcode       string `json:"qrcode"`
	Base64       string `json:"base64"`
	PairingCode  string `json:"pairingCode"`
	PairingSnake string `json:"pairing_code"`
}

func pickPairingCode(d qrData) string {
	if s := strings.TrimSpace(d.PairingCode); s != "" {
		return s
	}
	return strings.TrimSpace(d.PairingSnake)
}

// pickQRImage escolhe string utilizável como imagem do QR (Evolution v2 usa muitas vezes "base64";
// o campo "code" pode ser ref Baileys "2@..." — não é PNG).
func pickQRImage(d qrData) string {
	candidates := []string{
		strings.TrimSpace(d.Base64),
		strings.TrimSpace(d.Qrcode),
		strings.TrimSpace(d.Code),
	}
	for _, s := range candidates {
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, "2@") {
			continue
		}
		if strings.HasPrefix(s, "data:image") {
			return s
		}
		if looksLikeBase64ImagePayload(s) {
			return s
		}
	}
	return ""
}

func looksLikeBase64ImagePayload(s string) bool {
	if len(s) < 32 {
		return false
	}
	for i := 0; i < len(s); i++ {
		r := s[i]
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '+', r == '/', r == '=':
			continue
		default:
			return false
		}
	}
	return true
}

// NormalizeQRDataURLForBrowser prepara src de <img>: data URL só quando o payload é base64 de imagem.
func NormalizeQRDataURLForBrowser(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	if strings.HasPrefix(code, "data:") {
		return code
	}
	if looksLikeBase64ImagePayload(code) {
		return "data:image/png;base64," + code
	}
	return code
}

func decodeQRFromStruct(d qrData) (*ConnectResponse, bool) {
	img := pickQRImage(d)
	pair := pickPairingCode(d)
	if img == "" && pair == "" {
		return nil, false
	}
	return &ConnectResponse{
		Code:        img,
		PairingCode: pair,
	}, true
}

func decodeQRMapLoose(raw []byte) (*ConnectResponse, bool) {
	var root map[string]interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, false
	}
	inner := root
	if d, ok := root["data"].(map[string]interface{}); ok {
		inner = d
	}
	getS := func(m map[string]interface{}, keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k].(string); ok {
				if t := strings.TrimSpace(v); t != "" {
					return t
				}
			}
		}
		return ""
	}
	pc := getS(inner, "pairingCode", "PairingCode")
	if pc == "" {
		pc = getS(inner, "pairing_code")
	}
	qrcodeStr := getS(inner, "qrcode", "Qrcode", "qrCode")
	if qrcodeStr == "" {
		if m2, ok := inner["qrcode"].(map[string]interface{}); ok {
			for _, k := range []string{"base64", "code", "qrcode"} {
				if v, ok := m2[k].(string); ok && strings.TrimSpace(v) != "" {
					qrcodeStr = strings.TrimSpace(v)
					break
				}
			}
		}
	}
	d := qrData{
		Base64:      getS(inner, "base64", "Base64"),
		Qrcode:      qrcodeStr,
		Code:        getS(inner, "code", "Code"),
		PairingCode: pc,
	}
	return decodeQRFromStruct(d)
}

func decodeQR(raw []byte) (*ConnectResponse, error) {
	var direct qrData
	if err := json.Unmarshal(raw, &direct); err == nil {
		if out, ok := decodeQRFromStruct(direct); ok {
			return out, nil
		}
	}
	var wrap dataWrap
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Data) > 0 {
		var inner qrData
		if err := json.Unmarshal(wrap.Data, &inner); err == nil {
			if out, ok := decodeQRFromStruct(inner); ok {
				return out, nil
			}
		}
		if out, ok := decodeQRMapLoose(wrap.Data); ok {
			return out, nil
		}
	}
	if out, ok := decodeQRMapLoose(raw); ok {
		return out, nil
	}
	return nil, fmt.Errorf("decode qr response")
}

func decodeState(raw []byte) (string, error) {
	var wrap dataWrap
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Data) > 0 {
		var m map[string]interface{}
		if err := json.Unmarshal(wrap.Data, &m); err == nil {
			if connected, ok := m["connected"].(bool); ok {
				if connected {
					return "connected", nil
				}
				return "disconnected", nil
			}
			if jid, ok := m["jid"].(string); ok && strings.TrimSpace(jid) != "" {
				return "connected", nil
			}
		}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", err
	}
	if connected, ok := m["connected"].(bool); ok {
		if connected {
			return "connected", nil
		}
		return "disconnected", nil
	}
	return "unknown", nil
}

// ConnectResponse devolve QR/pair code.
type ConnectResponse struct {
	Code        string `json:"code"` // base64 QR
	PairingCode string `json:"pairingCode"`
	Count       int    `json:"count"`
}

func (c *EvolutionClient) ConnectInstance(ctx context.Context, instanceToken string) (*ConnectResponse, error) {
	token := strings.TrimSpace(instanceToken)
	if token == "" {
		token = c.apiKey
	}
	u := fmt.Sprintf("%s/instance/qr", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("evolution qr %d: %s", resp.StatusCode, string(raw))
	}
	return decodeQR(raw)
}

// ConnectionState GET /instance/status com token da instância.
func (c *EvolutionClient) ConnectionState(ctx context.Context, instanceToken string) (string, error) {
	token := strings.TrimSpace(instanceToken)
	if token == "" {
		token = c.apiKey
	}
	u := fmt.Sprintf("%s/instance/status", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("apikey", token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("evolution status %d: %s", resp.StatusCode, string(raw))
	}
	return decodeState(raw)
}

type setWebhookRequest struct {
	Enabled         bool              `json:"enabled"`
	URL             string            `json:"url"`
	WebhookByEvents bool              `json:"webhookByEvents"`
	WebhookBase64   bool              `json:"webhookBase64"`
	Events          []string          `json:"events"`
	Headers         map[string]string `json:"headers,omitempty"`
}

// SetInstanceWebhookOpts opções extra para POST /webhook/set/{instance}.
type SetInstanceWebhookOpts struct {
	// Headers enviados em cada POST do Evolution para a tua API (ex.: X-Webhook-Secret).
	Headers map[string]string
}

// evolutionGoConnectBody Evolution Go (docs.evolutionfoundation.com.br): POST /instance/connect
// com header instanceId = UUID da instância (não o nome técnico).
type evolutionGoConnectBody struct {
	WebhookURL string   `json:"webhookUrl"`
	Subscribe  []string `json:"subscribe"`
	Immediate  bool     `json:"immediate"`
}

// FindInstanceRemoteID devolve o campo `id` da listagem /instance/all para o nome da instância.
func (c *EvolutionClient) FindInstanceRemoteID(ctx context.Context, instanceName string) (string, error) {
	name := strings.TrimSpace(instanceName)
	if name == "" {
		return "", fmt.Errorf("instance name vazio")
	}
	list, err := c.FetchInstances(ctx)
	if err != nil {
		return "", err
	}
	want := strings.ToLower(name)
	for _, ins := range list {
		if strings.ToLower(strings.TrimSpace(ins.Name)) == want {
			id := strings.TrimSpace(ins.InstanceID)
			if id == "" {
				return "", fmt.Errorf("evolution: instância %q sem id remoto", name)
			}
			return id, nil
		}
	}
	return "", fmt.Errorf("evolution: instância %q não encontrada em /instance/all", name)
}

// setWebhookEvolutionGoConnect documentação Evolution Go — actualiza webhook no painel.
func (c *EvolutionClient) setWebhookEvolutionGoConnect(ctx context.Context, remoteInstanceID, webhookURL string) error {
	remoteInstanceID = strings.TrimSpace(remoteInstanceID)
	if remoteInstanceID == "" {
		return fmt.Errorf("instanceId remoto vazio")
	}
	u := c.baseURL + "/instance/connect"
	body, err := json.Marshal(evolutionGoConnectBody{
		WebhookURL: strings.TrimSpace(webhookURL),
		Subscribe:  []string{"ALL"},
		Immediate:  false,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("instanceId", remoteInstanceID)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("evolution go connect/webhook %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

func (c *EvolutionClient) setWebhookNodeAPI(ctx context.Context, instanceName, instanceToken, webhookURL string, opts *SetInstanceWebhookOpts) error {
	name := strings.TrimSpace(instanceName)
	if name == "" {
		return fmt.Errorf("instance name vazio")
	}
	u := fmt.Sprintf("%s/webhook/set/%s", c.baseURL, url.PathEscape(name))
	events := EvolutionWebhookDefaultEvents
	var hdr map[string]string
	if opts != nil && len(opts.Headers) > 0 {
		hdr = opts.Headers
	}
	body, err := json.Marshal(setWebhookRequest{
		Enabled:         true,
		URL:             strings.TrimSpace(webhookURL),
		WebhookByEvents: false,
		WebhookBase64:   true,
		Events:          events,
		Headers:         hdr,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	token := strings.TrimSpace(instanceToken)
	if token == "" {
		token = c.apiKey
	}
	req.Header.Set("apikey", token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("evolution set webhook %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

// SetInstanceWebhook configura webhook na Evolution.
// 1) Evolution Go (evoapicloud/evolution-go): POST /instance/connect + header instanceId (UUID) — é o que o manager usa.
// 2) Fallback Evolution API (Node): POST /webhook/set/{nome} com apikey = token da instância.
func (c *EvolutionClient) SetInstanceWebhook(ctx context.Context, instanceName, instanceToken, webhookURL string, opts *SetInstanceWebhookOpts) error {
	name := strings.TrimSpace(instanceName)
	if name == "" {
		return fmt.Errorf("instance name vazio")
	}
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return fmt.Errorf("webhook url vazio")
	}

	var goErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			t := time.NewTimer(350 * time.Millisecond)
			select {
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
			}
		}
		rid, ferr := c.FindInstanceRemoteID(ctx, name)
		if ferr != nil || rid == "" {
			goErr = ferr
			continue
		}
		if err := c.setWebhookEvolutionGoConnect(ctx, rid, webhookURL); err == nil {
			return nil
		} else {
			goErr = err
		}
	}

	if err := c.setWebhookNodeAPI(ctx, name, instanceToken, webhookURL, opts); err != nil {
		if goErr != nil {
			return fmt.Errorf("evolution go: %v; evolution node: %w", goErr, err)
		}
		return err
	}
	return nil
}

// DeleteRemoteInstance DELETE /instance/delete/{idOrName} com apikey global.
// Evolution Go exige UUID no path (repositório valida uuid.Parse); o nome lógico da instância
// é resolvido via FetchInstances. Evolution API (Node) costuma aceitar o nome no path — nesse caso
// usamos o nome quando a listagem falhar ou não devolver id.
// 404 é tratado como sucesso (rota/recurso inexistente).
func (c *EvolutionClient) DeleteRemoteInstance(ctx context.Context, instanceName string) error {
	name := strings.TrimSpace(instanceName)
	if name == "" {
		return fmt.Errorf("instance name vazio")
	}

	pathParam := name
	if _, err := uuid.Parse(pathParam); err != nil {
		list, listErr := c.FetchInstances(ctx)
		if listErr == nil {
			var remoteID string
			found := false
			for _, ins := range list {
				if strings.EqualFold(strings.TrimSpace(ins.Name), name) {
					found = true
					remoteID = strings.TrimSpace(ins.InstanceID)
					break
				}
			}
			if !found {
				return nil
			}
			if remoteID != "" {
				pathParam = remoteID
			}
		}
	}

	return c.deleteInstanceByPath(ctx, pathParam)
}

func (c *EvolutionClient) deleteInstanceByPath(ctx context.Context, pathSegment string) error {
	seg := strings.TrimSpace(pathSegment)
	if seg == "" {
		return fmt.Errorf("instance id/name vazio")
	}
	u := fmt.Sprintf("%s/instance/delete/%s", c.baseURL, url.PathEscape(seg))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("evolution delete instance %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}
