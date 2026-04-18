package service

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

// UpsertInboundMessage cria/atualiza conversa e mensagem a partir de webhook.
// remoteJid / remoteJidAlt: o WhatsApp pode mandar @lid no principal e o número em remoteJidAlt — procuramos conversa por qualquer chave normalizada.
// KeyID (Baileys): quando preenchido evita duplicar ao reprocessar webhooks.
// Devolve workspaceID, conversationID e o ID da mensagem inbound criada (Nil se ignorou duplicado).
func UpsertInboundMessage(db *gorm.DB, log *zap.Logger, evolutionInstanceName string, in InboundText) (workspaceID uuid.UUID, conversationID uuid.UUID, inboundMessageID uuid.UUID, _ error) {
	name := strings.ToLower(strings.TrimSpace(evolutionInstanceName))
	keys := CollectJIDLookupKeys(in.From, in.RemoteJidAlt)
	if name == "" || len(keys) == 0 {
		return uuid.Nil, uuid.Nil, uuid.Nil, nil
	}
	canonicalJID := keys[0]
	extID := strings.TrimSpace(in.KeyID)

	receivedAt := in.ReceivedAt
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}

	var inst model.WhatsAppInstance
	if err := db.Where("evolution_instance_name = ?", name).First(&inst).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Debug("webhook: instância desconhecida na API", zap.String("instance", name))
			return uuid.Nil, uuid.Nil, uuid.Nil, nil
		}
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}

	if extID != "" {
		var n int64
		if err := db.Model(&model.Message{}).
			Joins("INNER JOIN conversations ON conversations.id = messages.conversation_id").
			Where("conversations.whats_app_instance_id = ? AND messages.external_id = ?", inst.ID, extID).
			Count(&n).Error; err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, err
		}
		if n > 0 {
			return uuid.Nil, uuid.Nil, uuid.Nil, nil
		}
	}
	explicitName := strings.TrimSpace(in.PushName)
	displayName := explicitName
	if displayName == "" {
		displayName = DisplayNameFromJID(canonicalJID)
	}

	var conv model.Conversation
	var err error
	found := false
	for _, jid := range keys {
		err = db.Where("workspace_id = ? AND whats_app_instance_id = ? AND contact_j_id = ?",
			inst.WorkspaceID, inst.ID, jid).First(&conv).Error
		if err == nil {
			found = true
			break
		}
		if err != gorm.ErrRecordNotFound {
			return uuid.Nil, uuid.Nil, uuid.Nil, err
		}
	}

	if !found {
		conv = model.Conversation{
			WorkspaceID:        inst.WorkspaceID,
			WhatsAppInstanceID: inst.ID,
			ContactJID:         canonicalJID,
			ContactName:        displayName,
			LastMessageAt:      receivedAt,
			LastMessagePreview: truncatePreview(in.Text),
			Channel:            "whatsapp",
			CreatedAt:          receivedAt,
			UpdatedAt:          receivedAt,
		}
		if err := db.Create(&conv).Error; err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, err
		}
	} else {
		now := time.Now().UTC()
		updates := map[string]interface{}{
			"updated_at": now,
		}
		// Não regredir last_message_at ao reprocessar webhooks antigos (reconcile) fora de ordem.
		if receivedAt.After(conv.LastMessageAt) || conv.LastMessageAt.IsZero() {
			updates["last_message_at"] = receivedAt
			updates["last_message_preview"] = truncatePreview(in.Text)
		}
		if explicitName != "" {
			updates["contact_name"] = explicitName
		}
		if err := db.Model(&conv).Updates(updates).Error; err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, err
		}
	}

	if extID == "" {
		t0 := receivedAt.Add(-10 * time.Second)
		t1 := receivedAt.Add(10 * time.Second)
		var n int64
		if err := db.Model(&model.Message{}).
			Where("conversation_id = ? AND direction = ? AND body = ? AND created_at BETWEEN ? AND ?",
				conv.ID, "inbound", in.Text, t0, t1).
			Count(&n).Error; err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, err
		}
		if n > 0 {
			return uuid.Nil, uuid.Nil, uuid.Nil, nil
		}
	}

	mt := strings.TrimSpace(in.MessageType)
	if mt == "" {
		mt = InferMessageTypeFromBody(in.Text)
	}
	if mt == "" {
		mt = "text"
	}
	msg := model.Message{
		ConversationID:     conv.ID,
		Direction:          "inbound",
		Body:               in.Text,
		ExternalID:         extID,
		MessageType:        mt,
		FileName:           strings.TrimSpace(in.MediaFileName),
		MimeType:           strings.TrimSpace(in.MediaMimeType),
		MediaRemoteURL:     strings.TrimSpace(in.MediaRemoteURL),
		WaMediaMessageJSON: strings.TrimSpace(in.WaMediaMessageJSON),
		CreatedAt:          receivedAt,
	}
	if err2 := db.Create(&msg).Error; err2 != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err2
	}
	return inst.WorkspaceID, conv.ID, msg.ID, nil
}

// UpsertOutboundFromWebhook grava mensagem enviada por nós (fromMe) na conversa do contacto em remoteJid.
func UpsertOutboundFromWebhook(db *gorm.DB, log *zap.Logger, evolutionInstanceName string, in InboundText) (workspaceID uuid.UUID, conversationID uuid.UUID, _ error) {
	name := strings.ToLower(strings.TrimSpace(evolutionInstanceName))
	keys := CollectJIDLookupKeys(in.From, in.RemoteJidAlt)
	if name == "" || len(keys) == 0 {
		return uuid.Nil, uuid.Nil, nil
	}
	canonicalJID := keys[0]
	extID := strings.TrimSpace(in.KeyID)

	sentAt := in.ReceivedAt
	if sentAt.IsZero() {
		sentAt = time.Now().UTC()
	}

	var inst model.WhatsAppInstance
	if err := db.Where("evolution_instance_name = ?", name).First(&inst).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Debug("webhook: instância desconhecida na API", zap.String("instance", name))
			return uuid.Nil, uuid.Nil, nil
		}
		return uuid.Nil, uuid.Nil, err
	}

	if extID != "" {
		var n int64
		if err := db.Model(&model.Message{}).
			Joins("INNER JOIN conversations ON conversations.id = messages.conversation_id").
			Where("conversations.whats_app_instance_id = ? AND messages.external_id = ?", inst.ID, extID).
			Count(&n).Error; err != nil {
			return uuid.Nil, uuid.Nil, err
		}
		if n > 0 {
			return uuid.Nil, uuid.Nil, nil
		}
	}

	var conv model.Conversation
	var err error
	found := false
	for _, jid := range keys {
		err = db.Where("workspace_id = ? AND whats_app_instance_id = ? AND contact_j_id = ?",
			inst.WorkspaceID, inst.ID, jid).First(&conv).Error
		if err == nil {
			found = true
			break
		}
		if err != gorm.ErrRecordNotFound {
			return uuid.Nil, uuid.Nil, err
		}
	}

	if !found {
		display := DisplayNameFromJID(canonicalJID)
		conv = model.Conversation{
			WorkspaceID:        inst.WorkspaceID,
			WhatsAppInstanceID: inst.ID,
			ContactJID:         canonicalJID,
			ContactName:        display,
			LastMessageAt:      sentAt,
			LastMessagePreview: truncatePreview(in.Text),
			Channel:            "whatsapp",
			CreatedAt:          sentAt,
			UpdatedAt:          sentAt,
		}
		if err := db.Create(&conv).Error; err != nil {
			return uuid.Nil, uuid.Nil, err
		}
	} else {
		now := time.Now().UTC()
		updates := map[string]interface{}{
			"updated_at": now,
		}
		if sentAt.After(conv.LastMessageAt) || conv.LastMessageAt.IsZero() {
			updates["last_message_at"] = sentAt
			updates["last_message_preview"] = truncatePreview(in.Text)
		}
		if err := db.Model(&conv).Updates(updates).Error; err != nil {
			return uuid.Nil, uuid.Nil, err
		}
	}

	if extID == "" {
		t0 := sentAt.Add(-10 * time.Second)
		t1 := sentAt.Add(10 * time.Second)
		var n int64
		if err := db.Model(&model.Message{}).
			Where("conversation_id = ? AND direction = ? AND body = ? AND created_at BETWEEN ? AND ?",
				conv.ID, "outbound", in.Text, t0, t1).
			Count(&n).Error; err != nil {
			return uuid.Nil, uuid.Nil, err
		}
		if n > 0 {
			return uuid.Nil, uuid.Nil, nil
		}
	}

	mt := strings.TrimSpace(in.MessageType)
	if mt == "" {
		mt = InferMessageTypeFromBody(in.Text)
	}
	if mt == "" {
		mt = "text"
	}
	msg := model.Message{
		ConversationID: conv.ID,
		Direction:        "outbound",
		Body:             in.Text,
		ExternalID:       extID,
		MessageType:      mt,
		FileName:         strings.TrimSpace(in.MediaFileName),
		MimeType:         strings.TrimSpace(in.MediaMimeType),
		MediaRemoteURL:   strings.TrimSpace(in.MediaRemoteURL),
		CreatedAt:        sentAt,
	}
	if err := db.Create(&msg).Error; err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return inst.WorkspaceID, conv.ID, nil
}

// OutboundRecord campos para gravar mensagem enviada (texto ou mídia).
type OutboundRecord struct {
	Body         string
	ExternalID   string
	MessageType  string // vazio => text
	FileName     string
	MimeType     string
}

// RecordOutboundMessage grava mensagem de texto enviada (operador ou auto-reply).
func RecordOutboundMessage(db *gorm.DB, conversationID uuid.UUID, body, externalID string) error {
	_, err := RecordOutbound(db, conversationID, OutboundRecord{
		Body: body, ExternalID: externalID, MessageType: "text",
	})
	return err
}

// RecordOutbound grava mensagem enviada (texto ou mídia). Devolve o id da mensagem criada.
func RecordOutbound(db *gorm.DB, conversationID uuid.UUID, rec OutboundRecord) (uuid.UUID, error) {
	mt := strings.TrimSpace(rec.MessageType)
	if mt == "" {
		mt = "text"
	}
	now := time.Now()
	msg := model.Message{
		ConversationID: conversationID,
		Direction:      "outbound",
		Body:           rec.Body,
		ExternalID:     strings.TrimSpace(rec.ExternalID),
		MessageType:    mt,
		FileName:       strings.TrimSpace(rec.FileName),
		MimeType:       strings.TrimSpace(rec.MimeType),
		CreatedAt:      now,
	}
	if err := db.Create(&msg).Error; err != nil {
		return uuid.Nil, err
	}
	var conv model.Conversation
	if err := db.First(&conv, "id = ?", conversationID).Error; err != nil {
		return uuid.Nil, err
	}
	preview := rec.Body
	if strings.TrimSpace(preview) == "" && strings.TrimSpace(rec.FileName) != "" {
		preview = rec.FileName
	}
	if strings.TrimSpace(preview) == "" {
		preview = "[" + mt + "]"
	}
	err := db.Model(&conv).Updates(map[string]interface{}{
		"last_message_at":      now,
		"last_message_preview": truncatePreview(preview),
		"updated_at":           now,
	}).Error
	return msg.ID, err
}

// PersistPortalOutboundWebhook grava webhook sintético (messages.upsert) para envios feitos pela API do site.
// Assim, após "Excluir conversa" (apaga só messages), "Recuperar" (reconcile) volta a inserir as mensagens enviadas por ti.
func PersistPortalOutboundWebhook(db *gorm.DB, evolutionInstanceSlug, remoteJid, remoteJidAlt, body, keyID string) error {
	slug := strings.ToLower(strings.TrimSpace(evolutionInstanceSlug))
	body = strings.TrimSpace(body)
	rj := strings.TrimSpace(remoteJid)
	if slug == "" || body == "" || rj == "" {
		return nil
	}
	ts := float64(time.Now().Unix())
	keyObj := map[string]interface{}{
		"remoteJid": rj,
		"fromMe":    true,
	}
	if kid := strings.TrimSpace(keyID); kid != "" {
		keyObj["id"] = kid
	}
	if alt := strings.TrimSpace(remoteJidAlt); alt != "" {
		keyObj["remoteJidAlt"] = alt
	}
	inner := map[string]interface{}{
		"key":              keyObj,
		"message":          map[string]interface{}{"conversation": body},
		"messageTimestamp": ts,
	}
	dataBytes, err := json.Marshal(inner)
	if err != nil {
		return err
	}
	type webhookEnvelope struct {
		Event    string          `json:"event"`
		Instance string          `json:"instance"`
		Data     json.RawMessage `json:"data"`
	}
	raw, err := json.Marshal(webhookEnvelope{
		Event:    "messages.upsert",
		Instance: slug,
		Data:     dataBytes,
	})
	if err != nil {
		return err
	}
	canonical := InboundCanonicalJID(rj, remoteJidAlt)
	rec := model.WebhookMessage{
		InstanceID: slug,
		Event:      "messages.upsert",
		RemoteJID:  canonical,
		Direction:  "outbound",
		Body:       body,
		RawPayload: raw,
	}
	return db.Create(&rec).Error
}

// PersistPortalOutboundWebhookMedia webhook sintético com imageMessage / documentMessage / … para reconcile.
func PersistPortalOutboundWebhookMedia(db *gorm.DB, evolutionInstanceSlug, remoteJid, remoteJidAlt, keyID, msgType, caption, fileName string) error {
	slug := strings.ToLower(strings.TrimSpace(evolutionInstanceSlug))
	rj := strings.TrimSpace(remoteJid)
	mt := strings.ToLower(strings.TrimSpace(msgType))
	if slug == "" || rj == "" || mt == "" {
		return nil
	}
	cap := strings.TrimSpace(caption)
	fn := strings.TrimSpace(fileName)
	body := cap
	if body == "" && fn != "" {
		body = fn
	}
	if body == "" {
		switch mt {
		case "image":
			body = "[imagem]"
		case "video":
			body = "[vídeo]"
		case "audio":
			body = "[áudio]"
		case "document":
			body = "[documento]"
		default:
			body = "[mídia]"
		}
	}
	ts := float64(time.Now().Unix())
	keyObj := map[string]interface{}{
		"remoteJid": rj,
		"fromMe":    true,
	}
	if kid := strings.TrimSpace(keyID); kid != "" {
		keyObj["id"] = kid
	}
	if alt := strings.TrimSpace(remoteJidAlt); alt != "" {
		keyObj["remoteJidAlt"] = alt
	}
	var msgObj map[string]interface{}
	switch mt {
	case "image":
		im := map[string]interface{}{}
		if cap != "" {
			im["caption"] = cap
		}
		msgObj = map[string]interface{}{"imageMessage": im}
	case "video":
		vm := map[string]interface{}{}
		if cap != "" {
			vm["caption"] = cap
		}
		msgObj = map[string]interface{}{"videoMessage": vm}
	case "audio":
		msgObj = map[string]interface{}{"audioMessage": map[string]interface{}{}}
	case "document":
		dm := map[string]interface{}{}
		if fn != "" {
			dm["fileName"] = fn
		}
		if cap != "" {
			dm["caption"] = cap
		}
		msgObj = map[string]interface{}{"documentMessage": dm}
	default:
		msgObj = map[string]interface{}{"conversation": body}
	}
	inner := map[string]interface{}{
		"key":              keyObj,
		"message":          msgObj,
		"messageTimestamp": ts,
	}
	dataBytes, err := json.Marshal(inner)
	if err != nil {
		return err
	}
	type webhookEnvelope struct {
		Event    string          `json:"event"`
		Instance string          `json:"instance"`
		Data     json.RawMessage `json:"data"`
	}
	raw, err := json.Marshal(webhookEnvelope{
		Event:    "messages.upsert",
		Instance: slug,
		Data:     dataBytes,
	})
	if err != nil {
		return err
	}
	canonical := InboundCanonicalJID(rj, remoteJidAlt)
	rec := model.WebhookMessage{
		InstanceID: slug,
		Event:      "messages.upsert",
		RemoteJID:  canonical,
		Direction:  "outbound",
		Body:       body,
		RawPayload: raw,
	}
	return db.Create(&rec).Error
}

func truncatePreview(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 160 {
		return s
	}
	return s[:157] + "..."
}

// ReconcileWebhooksToInbox reprocessa webhook_messages (auditoria) e grava mensagens que faltem na conversa.
// bodyContains filtra por subcadeia no campo body (ex. "Legal cara"); vazio = últimos limit eventos da instância.
func ReconcileWebhooksToInbox(db *gorm.DB, log *zap.Logger, rdb *redis.Client, wid uuid.UUID, inst model.WhatsAppInstance, bodyContains string, limit int) (newMessages int, scanned int, err error) {
	if limit <= 0 || limit > 2000 {
		limit = 300
	}
	slug := strings.ToLower(strings.TrimSpace(inst.EvolutionInstanceName))
	q := db.Model(&model.WebhookMessage{}).
		Where("LOWER(TRIM(instance_id)) = ?", slug).
		Where("raw_payload IS NOT NULL").
		Order("created_at DESC").
		Limit(limit)
	if t := strings.TrimSpace(bodyContains); t != "" {
		q = q.Where("LOWER(body) LIKE ?", "%"+strings.ToLower(t)+"%")
	}
	var rows []model.WebhookMessage
	if err := q.Find(&rows).Error; err != nil {
		return 0, 0, err
	}
	scanned = len(rows)
	evoName := slug

	type recRow struct {
		wm      model.WebhookMessage
		inbound InboundText
	}
	var batch []recRow
	for _, row := range rows {
		var payload EvolutionWebhookPayload
		if err := json.Unmarshal(row.RawPayload, &payload); err != nil {
			continue
		}
		data := NormalizeWebhookData(payload.Data)
		inbound, ok := ParseInboundFromEvolution(payload.Event, data)
		if !ok {
			continue
		}
		canonical := InboundCanonicalJID(inbound.From, inbound.RemoteJidAlt)
		if canonical == "" || inbound.Text == "" {
			continue
		}
		if inbound.ReceivedAt.IsZero() {
			inbound.ReceivedAt = row.CreatedAt.UTC()
		}
		batch = append(batch, recRow{wm: row, inbound: inbound})
	}
	sort.Slice(batch, func(i, j int) bool {
		ti := batch[i].inbound.ReceivedAt
		tj := batch[j].inbound.ReceivedAt
		if ti.Equal(tj) {
			return batch[i].wm.CreatedAt.Before(batch[j].wm.CreatedAt)
		}
		return ti.Before(tj)
	})

	notify := map[uuid.UUID]struct{}{}
	for _, item := range batch {
		in := item.inbound
		var cwid, ccid uuid.UUID
		var err error
		if in.FromMe {
			cwid, ccid, err = UpsertOutboundFromWebhook(db, log, evoName, in)
		} else {
			cwid, ccid, _, err = UpsertInboundMessage(db, log, evoName, in)
		}
		if err != nil {
			log.Warn("reconcile upsert", zap.Error(err))
			continue
		}
		if cwid != uuid.Nil && ccid != uuid.Nil {
			newMessages++
			notify[ccid] = struct{}{}
		}
	}
	for cid := range notify {
		if err := RefreshConversationPreview(db, cid); err != nil {
			log.Debug("reconcile refresh preview", zap.String("conversation_id", cid.String()), zap.Error(err))
		}
	}
	if rdb != nil {
		for cid := range notify {
			PublishInboxEvent(rdb, wid, map[string]interface{}{
				"type":            "message.created",
				"payload":         map[string]string{"conversation_id": cid.String()},
				"conversation_id": cid.String(),
			})
		}
	}
	return newMessages, scanned, nil
}

// RefreshConversationPreview atualiza preview da conversa após import de histórico.
func RefreshConversationPreview(db *gorm.DB, conversationID uuid.UUID) error {
	var last model.Message
	if err := db.Where("conversation_id = ?", conversationID).Order("created_at DESC").First(&last).Error; err != nil {
		return err
	}
	now := time.Now()
	return db.Model(&model.Conversation{}).Where("id = ?", conversationID).Updates(map[string]interface{}{
		"last_message_at":      last.CreatedAt,
		"last_message_preview": truncatePreview(last.Body),
		"updated_at":           now,
	}).Error
}
