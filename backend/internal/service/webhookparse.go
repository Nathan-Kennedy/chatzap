package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func normEvent(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// IsInboundMessageEvent indica se o nome do evento costuma carregar payload de mensagem (para logs/diagnóstico).
func IsInboundMessageEvent(ev string) bool {
	switch normEvent(ev) {
	case "messages.upsert", "messages_upsert", "messagesupsert", "message", "send.message",
		"send_message", "send-message", "messages-send", "messages_send":
		return true
	default:
		return false
	}
}

// EvolutionWebhookPayload formato comum Evolution v2 (campos principais).
type EvolutionWebhookPayload struct {
	Event    string          `json:"event"`
	Instance string          `json:"instance"`
	APIKey   string          `json:"apikey,omitempty"`
	Data     json.RawMessage `json:"data"`
}

// NormalizeWebhookData expande `data` quando a Evolution envia JSON string + Base64 (webhookBase64).
func NormalizeWebhookData(raw json.RawMessage) json.RawMessage {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 {
		return raw
	}
	if b[0] == '{' || b[0] == '[' {
		return raw
	}
	if b[0] != '"' {
		return raw
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return raw
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return raw
	}
	dec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		dec, err = base64.RawURLEncoding.DecodeString(s)
	}
	if err != nil {
		// Algumas instalações enviam o JSON da mensagem como string JSON (sem Base64).
		t := strings.TrimSpace(s)
		if len(t) > 0 && (t[0] == '{' || t[0] == '[') {
			return json.RawMessage([]byte(t))
		}
		return raw
	}
	t := bytes.TrimSpace(dec)
	if len(t) == 0 || (t[0] != '{' && t[0] != '[') {
		return raw
	}
	return json.RawMessage(dec)
}

// InboundText resultado após interpretar messages.upsert.
type InboundText struct {
	From         string
	RemoteJidAlt string // JID alternativo (ex. PN quando remoteJid é @lid)
	Text         string
	FromMe       bool
	PushName     string // nome no WhatsApp (quando o webhook envia)
	ReceivedAt   time.Time
	KeyID        string // id da mensagem no key (Baileys); dedupe em replay/sync
	MessageType  string // text | image | video | audio | document | sticker
	MediaFileName string
	MediaMimeType string
	MediaRemoteURL string // URL http(s) no payload Baileys — GET attachment faz proxy
	// Evolution Go: JSON do nó message (proto) para POST /message/downloadmedia.
	WaMediaMessageJSON string
}

// InferMessageTypeFromBody infere tipo a partir de placeholders [imagem], etc.
func InferMessageTypeFromBody(body string) string {
	switch strings.TrimSpace(body) {
	case "[imagem]":
		return "image"
	case "[vídeo]", "[video]":
		return "video"
	case "[áudio]":
		return "audio"
	case "[documento]":
		return "document"
	case "[sticker]":
		return "sticker"
	default:
		return ""
	}
}

// mapByKeys devolve o primeiro map aninhado encontrado (Evolution Go costuma usar PascalCase no JSON).
func mapByKeys(data map[string]interface{}, keys ...string) map[string]interface{} {
	if data == nil {
		return nil
	}
	for _, k := range keys {
		if v, ok := data[k]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				return m
			}
		}
	}
	return nil
}

// strFromMap devolve a primeira string não vazia entre as chaves tentadas.
func strFromMap(m map[string]interface{}, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		if s, ok := m[k].(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return ""
}

// jidFromUserServerObject monta user@server quando o JSON traz RemoteJid como objeto (protobuf / Evolution Go).
func jidFromUserServerObject(v interface{}) string {
	m, ok := v.(map[string]interface{})
	if !ok {
		return ""
	}
	user := strFromMap(m, "user", "User")
	serv := strFromMap(m, "server", "Server")
	if user != "" && serv != "" {
		return user + "@" + serv
	}
	return ""
}

// jidFromFlexible lê JID como string ou como objeto {user, server}.
func jidFromFlexible(m map[string]interface{}, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		if s, ok := v.(string); ok {
			if t := strings.TrimSpace(s); t != "" {
				return t
			}
		}
		if j := jidFromUserServerObject(v); j != "" {
			return j
		}
	}
	return ""
}

// NormalizeEpochToTime interpreta messageTimestamp Baileys/Evolution: segundos Unix ou milissegundos (>1e12).
func NormalizeEpochToTime(v float64) time.Time {
	if v <= 0 {
		return time.Time{}
	}
	iv := int64(v)
	if iv > 1_000_000_000_000 {
		return time.UnixMilli(iv).UTC()
	}
	return time.Unix(iv, 0).UTC()
}

func parseMessageTimestamp(data map[string]interface{}) time.Time {
	if data == nil {
		return time.Time{}
	}
	for _, k := range []string{"messageTimestamp", "MessageTimestamp"} {
		raw, ok := data[k]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case float64:
			if t := NormalizeEpochToTime(v); !t.IsZero() {
				return t
			}
		case int:
			if t := NormalizeEpochToTime(float64(v)); !t.IsZero() {
				return t
			}
		case int64:
			if t := NormalizeEpochToTime(float64(v)); !t.IsZero() {
				return t
			}
		case json.Number:
			f, err := v.Float64()
			if err == nil {
				if t := NormalizeEpochToTime(f); !t.IsZero() {
					return t
				}
			}
		}
	}
	return time.Time{}
}

func envelopeHasWAMessageShape(m map[string]interface{}) bool {
	if m == nil {
		return false
	}
	if mapByKeys(m, "key", "Key") != nil {
		return true
	}
	if mapByKeys(m, "message", "Message") != nil {
		return true
	}
	for _, k := range []string{"messages", "Messages"} {
		if x, ok := m[k].([]interface{}); ok && len(x) > 0 {
			return true
		}
	}
	return false
}

// unwrapEvolutionDataEnvelope segue cascas { "data": { "key", "message" } } que a Evolution às vezes envolve.
func unwrapEvolutionDataEnvelope(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	for depth := 0; depth < 8; depth++ {
		if envelopeHasWAMessageShape(m) {
			return m
		}
		var inner map[string]interface{}
		var ok bool
		if inner, ok = m["data"].(map[string]interface{}); !ok {
			inner, ok = m["Data"].(map[string]interface{})
		}
		if !ok {
			return m
		}
		m = inner
	}
	return m
}

// ParseInboundFromEvolution extrai remetente e texto de eventos messages.upsert.
func ParseInboundFromEvolution(event string, data json.RawMessage) (InboundText, bool) {
	if !IsInboundMessageEvent(event) {
		return InboundText{}, false
	}

	trim := bytes.TrimSpace(data)
	if len(trim) > 0 && trim[0] == '[' {
		var arr []interface{}
		if err := json.Unmarshal(data, &arr); err != nil {
			return InboundText{}, false
		}
		if len(arr) == 0 {
			return InboundText{}, false
		}
		if m0, ok := arr[0].(map[string]interface{}); ok {
			return extractFromMessageMap(unwrapEvolutionDataEnvelope(m0))
		}
		return InboundText{}, false
	}

	var asObj map[string]interface{}
	if err := json.Unmarshal(data, &asObj); err != nil {
		return InboundText{}, false
	}
	asObj = unwrapEvolutionDataEnvelope(asObj)

	// data pode ser mensagem única ou { "messages": [ ... ] } (Evolution Go pode usar "Messages").
	var msgs []interface{}
	if m, ok := asObj["messages"].([]interface{}); ok && len(m) > 0 {
		msgs = m
	} else if m, ok := asObj["Messages"].([]interface{}); ok && len(m) > 0 {
		msgs = m
	}
	if len(msgs) > 0 {
		if m0, ok := msgs[0].(map[string]interface{}); ok {
			return extractFromMessageMap(unwrapEvolutionDataEnvelope(m0))
		}
		return InboundText{}, false
	}

	return extractFromMessageMap(asObj)
}

// WebhookMessageTextPreview extrai só texto do payload (diagnóstico quando falta JID).
func WebhookMessageTextPreview(event string, data json.RawMessage) string {
	if !IsInboundMessageEvent(event) {
		return ""
	}
	data = NormalizeWebhookData(data)
	trim := bytes.TrimSpace(data)
	if len(trim) > 0 && trim[0] == '[' {
		var arr []interface{}
		if json.Unmarshal(data, &arr) != nil || len(arr) == 0 {
			return ""
		}
		if m0, ok := arr[0].(map[string]interface{}); ok {
			return previewTextFromMessageMap(unwrapEvolutionDataEnvelope(m0))
		}
		return ""
	}
	var asObj map[string]interface{}
	if json.Unmarshal(data, &asObj) != nil {
		return ""
	}
	asObj = unwrapEvolutionDataEnvelope(asObj)
	var msgs []interface{}
	if m, ok := asObj["messages"].([]interface{}); ok && len(m) > 0 {
		msgs = m
	} else if m, ok := asObj["Messages"].([]interface{}); ok && len(m) > 0 {
		msgs = m
	}
	if len(msgs) > 0 {
		if m0, ok := msgs[0].(map[string]interface{}); ok {
			return previewTextFromMessageMap(unwrapEvolutionDataEnvelope(m0))
		}
		return ""
	}
	return previewTextFromMessageMap(asObj)
}

func previewTextFromMessageMap(data map[string]interface{}) string {
	msg := mapByKeys(data, "message", "Message")
	if t := textFromBaileysMessage(msg); t != "" {
		return t
	}
	if inner := mapByKeys(msg, "message", "Message"); inner != nil {
		if t := textFromBaileysMessage(inner); t != "" {
			return t
		}
	}
	return strFromMap(data, "body", "Body", "text", "Text", "content", "Content")
}

func truthyBoolFromMap(m map[string]interface{}, keys ...string) bool {
	if m == nil {
		return false
	}
	for _, k := range keys {
		if fm, ok := m[k].(bool); ok && fm {
			return true
		}
		if n, ok := m[k].(float64); ok && n != 0 {
			return true
		}
		if s, ok := m[k].(string); ok {
			if strings.EqualFold(strings.TrimSpace(s), "true") || strings.TrimSpace(s) == "1" {
				return true
			}
		}
	}
	return false
}

func keyFromMeTrue(key map[string]interface{}) bool {
	return truthyBoolFromMap(key, "fromMe", "FromMe")
}

// unwrapBaileysMessageContent segue wrappers comuns (ephemeral, viewOnce, etc.).
func unwrapBaileysMessageContent(msg map[string]interface{}) map[string]interface{} {
	if msg == nil {
		return nil
	}
	if inner := mapByKeys(msg, "editedMessage", "EditedMessage"); inner != nil {
		if m := mapByKeys(inner, "message", "Message"); m != nil {
			return unwrapBaileysMessageContent(m)
		}
	}
	if inner := mapByKeys(msg, "ephemeralMessage", "EphemeralMessage"); inner != nil {
		if m := mapByKeys(inner, "message", "Message"); m != nil {
			return unwrapBaileysMessageContent(m)
		}
	}
	if inner := mapByKeys(msg, "viewOnceMessage", "ViewOnceMessage"); inner != nil {
		if m := mapByKeys(inner, "message", "Message"); m != nil {
			return unwrapBaileysMessageContent(m)
		}
	}
	if inner := mapByKeys(msg, "documentWithCaptionMessage", "DocumentWithCaptionMessage"); inner != nil {
		if m := mapByKeys(inner, "message", "Message"); m != nil {
			return unwrapBaileysMessageContent(m)
		}
	}
	return msg
}

func firstHTTPURLFromMap(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	for _, k := range []string{"url", "Url", "URL", "mediaUrl", "MediaUrl"} {
		if s := strFromMap(m, k); s != "" {
			if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
				return s
			}
		}
	}
	return ""
}

func mediaMetaFromBaileysMessage(msg map[string]interface{}) (kind, url, fileName, mime string) {
	msg = unwrapBaileysMessageContent(msg)
	if msg == nil {
		return "", "", "", ""
	}
	pairs := []struct {
		keys []string
		kind string
	}{
		{[]string{"imageMessage", "ImageMessage"}, "image"},
		{[]string{"videoMessage", "VideoMessage"}, "video"},
		{[]string{"audioMessage", "AudioMessage"}, "audio"},
		{[]string{"pttMessage", "PttMessage"}, "audio"},
		{[]string{"documentMessage", "DocumentMessage"}, "document"},
		{[]string{"stickerMessage", "StickerMessage"}, "sticker"},
	}
	for _, p := range pairs {
		sub := mapByKeys(msg, p.keys...)
		if sub == nil {
			continue
		}
		url = firstHTTPURLFromMap(sub)
		fileName = strFromMap(sub, "fileName", "FileName", "filename", "Filename")
		mime = strFromMap(sub, "mimetype", "Mimetype", "mimeType", "MimeType")
		if fileName == "" && p.kind == "document" {
			fileName = strFromMap(sub, "title", "Title")
		}
		return p.kind, url, fileName, mime
	}
	return "", "", "", ""
}

func extractMediaMetaFromMessageWrapper(msg map[string]interface{}) (kind, url, fileName, mime string) {
	kind, url, fileName, mime = mediaMetaFromBaileysMessage(msg)
	if kind == "" && msg != nil {
		if inner := mapByKeys(msg, "message", "Message"); inner != nil {
			kind, url, fileName, mime = mediaMetaFromBaileysMessage(inner)
		}
	}
	return kind, url, fileName, mime
}

func placeholderForMediaKind(kind string) string {
	switch kind {
	case "image":
		return "[imagem]"
	case "video":
		return "[vídeo]"
	case "audio":
		return "[áudio]"
	case "document":
		return "[documento]"
	case "sticker":
		return "[sticker]"
	default:
		return "[mídia]"
	}
}

func textFromBaileysMessage(msg map[string]interface{}) string {
	msg = unwrapBaileysMessageContent(msg)
	if msg == nil {
		return ""
	}
	if t := strFromMap(msg, "conversation", "Conversation"); t != "" {
		return t
	}
	if et := mapByKeys(msg, "extendedTextMessage", "ExtendedTextMessage"); et != nil {
		if t := strFromMap(et, "text", "Text"); t != "" {
			return t
		}
	}
	type msgCap struct {
		msgKeys []string
		capKeys []string
	}
	for _, mc := range []msgCap{
		{[]string{"imageMessage", "ImageMessage"}, []string{"caption", "Caption"}},
		{[]string{"videoMessage", "VideoMessage"}, []string{"caption", "Caption"}},
		{[]string{"documentMessage", "DocumentMessage"}, []string{"caption", "Caption"}},
	} {
		if m := mapByKeys(msg, mc.msgKeys...); m != nil {
			if t := strFromMap(m, mc.capKeys...); t != "" {
				return t
			}
		}
	}
	if b := mapByKeys(msg, "buttonsResponseMessage", "ButtonsResponseMessage"); b != nil {
		if t := strFromMap(b, "selectedDisplayText", "SelectedDisplayText"); t != "" {
			return t
		}
	}
	if l := mapByKeys(msg, "listResponseMessage", "ListResponseMessage"); l != nil {
		if t := strFromMap(l, "title", "Title"); t != "" {
			return t
		}
		if s := mapByKeys(l, "singleSelectReply", "SingleSelectReply"); s != nil {
			if t := strFromMap(s, "selectedRowId", "SelectedRowId"); t != "" {
				return t
			}
		}
	}
	if tm := mapByKeys(msg, "templateMessage", "TemplateMessage"); tm != nil {
		if ht := mapByKeys(tm, "hydratedTemplate", "HydratedTemplate"); ht != nil {
			if t := strFromMap(ht, "hydratedContentText", "HydratedContentText", "hydratedTitleText", "HydratedTitleText"); t != "" {
				return t
			}
		}
		if t := strFromMap(tm, "title", "Title"); t != "" {
			return t
		}
	}
	if bm := mapByKeys(msg, "buttonsMessage", "ButtonsMessage"); bm != nil {
		if t := strFromMap(bm, "contentText", "ContentText", "text", "Text"); t != "" {
			return t
		}
	}
	if im := mapByKeys(msg, "interactiveMessage", "InteractiveMessage"); im != nil {
		if b := mapByKeys(im, "body", "Body"); b != nil {
			if t := strFromMap(b, "text", "Text"); t != "" {
				return t
			}
		}
		if t := strFromMap(im, "contentText", "ContentText"); t != "" {
			return t
		}
	}
	if mapByKeys(msg, "locationMessage", "LocationMessage") != nil {
		return "[localização]"
	}
	if mapByKeys(msg, "liveLocationMessage", "LiveLocationMessage") != nil {
		return "[localização ao vivo]"
	}
	if mapByKeys(msg, "contactMessage", "ContactMessage") != nil {
		return "[contacto]"
	}
	// Mídia sem legenda — comum em envios pelo telefone / Business; placeholder evita perder o evento.
	for _, pair := range []struct {
		keys []string
		tag  string
	}{
		{[]string{"imageMessage", "ImageMessage"}, "[imagem]"},
		{[]string{"videoMessage", "VideoMessage"}, "[vídeo]"},
		{[]string{"documentMessage", "DocumentMessage"}, "[documento]"},
		{[]string{"stickerMessage", "StickerMessage"}, "[sticker]"},
		{[]string{"audioMessage", "AudioMessage"}, "[áudio]"},
		{[]string{"pttMessage", "PttMessage"}, "[áudio]"},
	} {
		if mapByKeys(msg, pair.keys...) != nil {
			return pair.tag
		}
	}
	return ""
}

func jidAltFromKey(key map[string]interface{}) string {
	if key == nil {
		return ""
	}
	return jidFromFlexible(key, "remoteJidAlt", "RemoteJidAlt")
}

// keyIDFromKeyMap lê key.id como string (Baileys); alguns payloads JSON trazem número ou json.Number.
func keyIDFromKeyMap(key map[string]interface{}) string {
	if key == nil {
		return ""
	}
	for _, k := range []string{"id", "Id", "ID"} {
		v, ok := key[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			if s := strings.TrimSpace(t); s != "" {
				return s
			}
		case json.Number:
			if s := strings.TrimSpace(t.String()); s != "" {
				return s
			}
		case float64:
			if t >= 0 && t == float64(int64(t)) {
				return strconv.FormatInt(int64(t), 10)
			}
		case int:
			return strconv.Itoa(t)
		case int64:
			return strconv.FormatInt(t, 10)
		}
	}
	return ""
}

// normalizeWaMessageFragmentForEvolution devolve o fragmento tipo proto Message (ex.: audioMessage)
// sem cascas extra {"message":...} que quebram POST /message/downloadmedia na Evolution.
func normalizeWaMessageFragmentForEvolution(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := unwrapBaileysMessageContent(m)
	for depth := 0; depth < 5; depth++ {
		kind, _, _, _ := mediaMetaFromBaileysMessage(out)
		if kind != "" {
			return out
		}
		inner := mapByKeys(out, "message", "Message")
		if inner == nil {
			return out
		}
		kind2, _, _, _ := mediaMetaFromBaileysMessage(inner)
		if kind2 != "" {
			return inner
		}
		out = inner
	}
	return out
}

func extractFromMessageMap(data map[string]interface{}) (InboundText, bool) {
	msg := mapByKeys(data, "message", "Message")
	key := mapByKeys(data, "key", "Key")
	// Key por vezes vem só dentro de Message (payloads Evolution / Baileys JSON).
	if key == nil {
		key = mapByKeys(msg, "key", "Key")
	}

	from := jidFromFlexible(key, "remoteJid", "RemoteJid", "remote_jid", "Remote_Jid")
	if from == "" {
		from = jidFromFlexible(key, "participant", "Participant")
	}

	info := mapByKeys(data, "info", "Info", "webhookData", "WebhookData")
	if from == "" && info != nil {
		from = jidFromFlexible(info, "remoteJid", "RemoteJid", "remote_jid", "Sender", "sender", "jid", "Jid", "chatId", "ChatId")
	}
	if from == "" {
		from = jidFromFlexible(data, "remoteJid", "RemoteJid", "remote_jid", "jid", "Jid", "senderJid", "SenderJid", "chatId", "ChatId", "from", "From")
	}

	keyID := keyIDFromKeyMap(key)
	alt := jidAltFromKey(key)
	if alt == "" && info != nil {
		alt = jidFromFlexible(info, "remoteJidAlt", "RemoteJidAlt")
	}

	push := strFromMap(data, "pushName", "PushName")
	if push == "" && info != nil {
		push = strFromMap(info, "pushName", "PushName")
	}

	ts := parseMessageTimestamp(data)
	if ts.IsZero() && info != nil {
		ts = parseMessageTimestamp(info)
	}

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
		text = strFromMap(data, "body", "Body", "text", "Text", "content", "Content")
	}
	if text == "" {
		return InboundText{}, false
	}
	if from == "" {
		// Sem JID não há conversa estável na inbox (InboundCanonicalJID falha).
		return InboundText{}, false
	}

	fromMe := keyFromMeTrue(key)
	if !fromMe {
		fromMe = truthyBoolFromMap(data, "fromMe", "FromMe")
	}
	msgType := strings.TrimSpace(mediaKind)
	if msgType == "" {
		msgType = InferMessageTypeFromBody(text)
	}
	if msgType == "" {
		msgType = "text"
	}
	var waMediaJSON string
	if msgType != "text" && msg != nil {
		storeMsg := unwrapBaileysMessageContent(msg)
		if storeMsg == nil {
			storeMsg = msg
		}
		storeMsg = normalizeWaMessageFragmentForEvolution(storeMsg)
		const maxWaJSON = 512 * 1024
		if storeMsg != nil {
			if b, err := json.Marshal(storeMsg); err == nil && len(b) > 0 && len(b) < maxWaJSON {
				waMediaJSON = string(b)
			}
		}
	}
	return InboundText{
		From:               from,
		RemoteJidAlt:       alt,
		Text:               text,
		FromMe:             fromMe,
		PushName:           push,
		ReceivedAt:         ts,
		KeyID:              keyID,
		MessageType:        msgType,
		MediaFileName:      mediaFN,
		MediaMimeType:      mediaMT,
		MediaRemoteURL:     mediaURL,
		WaMediaMessageJSON: waMediaJSON,
	}, true
}
