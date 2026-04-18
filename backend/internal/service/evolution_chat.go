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
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

// ChatFindMessages POST /chat/findMessages/{instance} (Evolution API v2). Pode devolver 404 em forks sem Chat Controller.
func (c *EvolutionClient) ChatFindMessages(ctx context.Context, instanceName, instanceToken, remoteJID string) (status int, raw []byte, err error) {
	name := strings.TrimSpace(instanceName)
	if name == "" {
		return 0, nil, fmt.Errorf("instance name vazio")
	}
	u := fmt.Sprintf("%s/chat/findMessages/%s", c.baseURL, url.PathEscape(name))
	body, err := json.Marshal(map[string]interface{}{
		"where": map[string]interface{}{
			"key": map[string]string{"remoteJid": remoteJID},
		},
	})
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	token := strings.TrimSpace(instanceToken)
	if token == "" {
		token = c.apiKey
	}
	req.Header.Set("apikey", token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ = io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, raw, nil
}

// HistoryImportItem mensagem normalizada para gravar em messages (alinhado ao parse de webhook / Inbox).
type HistoryImportItem struct {
	ExternalID     string
	Body           string
	Direction      string // inbound | outbound
	CreatedAt      time.Time
	MessageType    string
	FileName       string
	MimeType       string
	MediaRemoteURL string
}

// ParseFindMessagesResponse extrai mensagens de várias formas de envelope JSON (Evolution / Baileys).
func ParseFindMessagesResponse(raw []byte) ([]HistoryImportItem, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var root interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	var msgs []interface{}
	switch v := root.(type) {
	case []interface{}:
		msgs = v
	case map[string]interface{}:
		if m, ok := extractMessagesArray(v); ok {
			msgs = m
		}
	}
	out := make([]HistoryImportItem, 0, len(msgs))
	for _, it := range msgs {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		if item, ok := historyFromBaileysMap(m); ok {
			out = append(out, item)
		}
	}
	return out, nil
}

func extractMessagesArray(v map[string]interface{}) ([]interface{}, bool) {
	if x, ok := v["messages"].([]interface{}); ok {
		return x, true
	}
	if d, ok := v["data"].(map[string]interface{}); ok {
		if x, ok := d["messages"].([]interface{}); ok {
			return x, true
		}
	}
	return nil, false
}

func historyFromBaileysMap(m map[string]interface{}) (HistoryImportItem, bool) {
	key := mapByKeys(m, "key", "Key")
	fromMe := keyFromMeTrue(key)
	extID := strFromMap(key, "id", "Id", "ID")
	dir := "inbound"
	if fromMe {
		dir = "outbound"
	}
	msg := mapByKeys(m, "message", "Message")
	text := textFromBaileysMessage(msg)
	if text == "" {
		if inner := mapByKeys(msg, "message", "Message"); inner != nil {
			text = textFromBaileysMessage(inner)
		}
	}
	mediaKind, mediaURL, mediaFN, mediaMT := extractMediaMetaFromMessageWrapper(msg)
	if text == "" && mediaKind != "" {
		text = placeholderForMediaKind(mediaKind)
	}
	if text == "" {
		return HistoryImportItem{}, false
	}
	ts := parseMessageTimestamp(m)
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	mt := strings.TrimSpace(mediaKind)
	if mt == "" {
		mt = InferMessageTypeFromBody(text)
	}
	if mt == "" {
		mt = "text"
	}
	return HistoryImportItem{
		ExternalID:     extID,
		Body:           text,
		Direction:      dir,
		CreatedAt:      ts,
		MessageType:    mt,
		FileName:       strings.TrimSpace(mediaFN),
		MimeType:       strings.TrimSpace(mediaMT),
		MediaRemoteURL: strings.TrimSpace(mediaURL),
	}, true
}

func isMediaPlaceholderBody(s string) bool {
	switch strings.TrimSpace(s) {
	case "[imagem]", "[vídeo]", "[video]", "[áudio]", "[documento]", "[sticker]", "[mídia]":
		return true
	default:
		return false
	}
}

// bodiesLikelySameMedia trata legenda vs placeholder ([imagem]) como a mesma mensagem (caso típico Recuperar + envio pela Inbox).
func bodiesLikelySameMedia(dbBody, importBody string) bool {
	a, b := strings.TrimSpace(dbBody), strings.TrimSpace(importBody)
	if a == b {
		return true
	}
	pa, pb := isMediaPlaceholderBody(a), isMediaPlaceholderBody(b)
	if pa && pb {
		return true
	}
	if pa != pb {
		caption := a
		if pa {
			caption = b
		}
		if strings.TrimSpace(caption) != "" {
			return true
		}
	}
	return false
}

func messageHasRenderableMedia(m *model.Message) bool {
	return strings.TrimSpace(m.StoredMediaPath) != "" || strings.TrimSpace(m.MediaRemoteURL) != ""
}

// mergeHistoryMetadataIntoExisting preenche URL/nome/ficheiro no registo já gravado pela Inbox; nunca apaga stored_media_path.
func mergeHistoryMetadataIntoExisting(db *gorm.DB, ex *model.Message, item HistoryImportItem) error {
	updates := map[string]interface{}{}
	if strings.TrimSpace(ex.MediaRemoteURL) == "" && strings.TrimSpace(item.MediaRemoteURL) != "" {
		updates["media_remote_url"] = strings.TrimSpace(item.MediaRemoteURL)
	}
	if strings.TrimSpace(ex.FileName) == "" && strings.TrimSpace(item.FileName) != "" {
		updates["file_name"] = strings.TrimSpace(item.FileName)
	}
	if strings.TrimSpace(ex.MimeType) == "" && strings.TrimSpace(item.MimeType) != "" {
		updates["mime_type"] = strings.TrimSpace(item.MimeType)
	}
	if strings.TrimSpace(ex.ExternalID) == "" && strings.TrimSpace(item.ExternalID) != "" {
		updates["external_id"] = strings.TrimSpace(item.ExternalID)
	}
	if isMediaPlaceholderBody(ex.Body) && !isMediaPlaceholderBody(item.Body) && strings.TrimSpace(item.Body) != "" {
		updates["body"] = strings.TrimSpace(item.Body)
	}
	if len(updates) == 0 {
		return nil
	}
	return db.Model(&model.Message{}).Where("id = ?", ex.ID).Updates(updates).Error
}

// InsertHistoryMessageIfNew grava mensagem importada (findMessages). Evita “fantasma” sem ficheiro quando a Inbox já gravou a mesma mídia com stored_media_path.
func InsertHistoryMessageIfNew(db *gorm.DB, conversationID uuid.UUID, item HistoryImportItem) (inserted bool, err error) {
	if item.Body == "" {
		return false, nil
	}
	var conv model.Conversation
	if err := db.First(&conv, "id = ?", conversationID).Error; err != nil {
		return false, err
	}
	instID := conv.WhatsAppInstanceID
	itemMT := strings.TrimSpace(item.MessageType)
	if itemMT == "" {
		itemMT = InferMessageTypeFromBody(item.Body)
	}
	if itemMT == "" {
		itemMT = "text"
	}

	// A) Mesmo external_id na instância: só enriquecer — nunca substituir linha com ficheiro local.
	if strings.TrimSpace(item.ExternalID) != "" {
		var existing model.Message
		err := db.Model(&model.Message{}).
			Joins("INNER JOIN conversations ON conversations.id = messages.conversation_id").
			Where("conversations.whats_app_instance_id = ? AND messages.external_id = ?", instID, item.ExternalID).
			First(&existing).Error
		if err == nil {
			if err := mergeHistoryMetadataIntoExisting(db, &existing, item); err != nil {
				return false, err
			}
			return false, nil
		}
		if err != gorm.ErrRecordNotFound {
			return false, err
		}
	}

	// B) findMessages muitas vezes devolve mídia sem URL: não inserir segunda linha se já existe uma servível no mesmo intervalo (legenda vs [imagem]).
	weakImport := itemMT != "text" && strings.TrimSpace(item.MediaRemoteURL) == ""
	if weakImport {
		var candidates []model.Message
		t0 := item.CreatedAt.Add(-90 * time.Second)
		t1 := item.CreatedAt.Add(90 * time.Second)
		if err := db.Where("conversation_id = ? AND direction = ? AND message_type = ? AND created_at BETWEEN ? AND ?",
			conversationID, item.Direction, itemMT, t0, t1).
			Find(&candidates).Error; err != nil {
			return false, err
		}
		for i := range candidates {
			ex := &candidates[i]
			if !messageHasRenderableMedia(ex) {
				continue
			}
			if bodiesLikelySameMedia(ex.Body, item.Body) {
				if err := mergeHistoryMetadataIntoExisting(db, ex, item); err != nil {
					return false, err
				}
				return false, nil
			}
		}
	}

	// C) dedupe estrito (texto / mesma linha exata)
	if item.ExternalID != "" {
		var n int64
		if err := db.Model(&model.Message{}).
			Joins("INNER JOIN conversations ON conversations.id = messages.conversation_id").
			Where("conversations.whats_app_instance_id = ? AND messages.external_id = ?", instID, item.ExternalID).
			Count(&n).Error; err != nil {
			return false, err
		}
		if n > 0 {
			return false, nil
		}
	} else {
		t0 := item.CreatedAt.Add(-12 * time.Second)
		t1 := item.CreatedAt.Add(12 * time.Second)
		var n int64
		if err := db.Model(&model.Message{}).
			Where("conversation_id = ? AND direction = ? AND body = ? AND created_at BETWEEN ? AND ?",
				conversationID, item.Direction, item.Body, t0, t1).
			Count(&n).Error; err != nil {
			return false, err
		}
		if n > 0 {
			return false, nil
		}
	}

	msg := model.Message{
		ConversationID: conversationID,
		Direction:        item.Direction,
		Body:             item.Body,
		ExternalID:       item.ExternalID,
		MessageType:      itemMT,
		FileName:         strings.TrimSpace(item.FileName),
		MimeType:         strings.TrimSpace(item.MimeType),
		MediaRemoteURL:   strings.TrimSpace(item.MediaRemoteURL),
		CreatedAt:        item.CreatedAt,
	}
	if err := db.Create(&msg).Error; err != nil {
		return false, err
	}
	return true, nil
}
