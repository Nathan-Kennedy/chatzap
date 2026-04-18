package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// EvolutionWhatsAppContact corresponde ao JSON de GET /user/contacts na Evolution Go.
type EvolutionWhatsAppContact struct {
	Jid          string `json:"Jid"`
	Found        bool   `json:"Found"`
	FirstName    string `json:"FirstName"`
	FullName     string `json:"FullName"`
	PushName     string `json:"PushName"`
	BusinessName string `json:"BusinessName"`
}

// ContactDisplayNameFromEvolution escolhe o melhor nome legível devolvido pela Evolution.
func ContactDisplayNameFromEvolution(c EvolutionWhatsAppContact) string {
	for _, s := range []string{c.BusinessName, c.FullName, c.PushName, c.FirstName} {
		if t := strings.TrimSpace(s); t != "" {
			return t
		}
	}
	return ""
}

// FetchWhatsAppContacts GET /user/contacts com apikey = token da instância (rota autenticada por instância).
func (c *EvolutionClient) FetchWhatsAppContacts(ctx context.Context, instanceToken string) ([]EvolutionWhatsAppContact, error) {
	u := fmt.Sprintf("%s/user/contacts", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(instanceToken)
	if token == "" {
		token = c.apiKey
	}
	req.Header.Set("apikey", token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("evolution user/contacts %d: %s", resp.StatusCode, string(raw))
	}
	var wrap struct {
		Data []EvolutionWhatsAppContact `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	return wrap.Data, nil
}
