package service

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

const (
	// DefaultHistoryMaxMessages quantas mensagens no máximo (mais recentes).
	DefaultHistoryMaxMessages = 45
	// DefaultHistoryMaxRunes teto de caracteres do bloco de histórico (contexto do modelo).
	DefaultHistoryMaxRunes = 16000
)

// BuildWhatsAppHistoryForLLM monta texto com mensagens anteriores da conversa.
// Se excludeMessageID != Nil, remove essa linha do histórico (mensagem atual por ID — evita deixar "Cliente: [áudio]"
// quando o body na BD ainda não coincide com currentInboundText após transcrição).
// Caso contrário, exclui a última inbound se o body for igual a currentInboundText.
func BuildWhatsAppHistoryForLLM(db *gorm.DB, conversationID uuid.UUID, currentInboundText string, maxMsgs int, maxRunes int, excludeMessageID uuid.UUID) (string, error) {
	if maxMsgs < 1 {
		maxMsgs = DefaultHistoryMaxMessages
	}
	if maxRunes < 400 {
		maxRunes = DefaultHistoryMaxRunes
	}

	var msgs []model.Message
	q := db.Where("conversation_id = ?", conversationID).Order("created_at DESC").Limit(maxMsgs + 8)
	if err := q.Find(&msgs).Error; err != nil {
		return "", err
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	if excludeMessageID != uuid.Nil {
		out := msgs[:0]
		for _, m := range msgs {
			if m.ID != excludeMessageID {
				out = append(out, m)
			}
		}
		msgs = out
	} else {
		cur := strings.TrimSpace(currentInboundText)
		if len(msgs) > 0 {
			last := msgs[len(msgs)-1]
			if last.Direction == "inbound" && strings.TrimSpace(last.Body) == cur {
				msgs = msgs[:len(msgs)-1]
			}
		}
	}

	var conv model.Conversation
	contactName := ""
	if err := db.Select("contact_name").First(&conv, "id = ?", conversationID).Error; err == nil {
		contactName = strings.TrimSpace(conv.ContactName)
	}
	if len(msgs) == 0 && contactName == "" {
		return "", nil
	}

	var b strings.Builder
	b.WriteString("Histórico desta conversa no WhatsApp (do mais antigo ao mais recente). ")
	b.WriteString("Podes usar morada, telefone e outros dados já ditos; não precisas citar o nome do cliente em toda resposta — só quando fizer sentido (saudação, proposta, confirmação de identidade). ")
	b.WriteString("Não inventes apelidos ou diminutivos do nome que o cliente não tenha usado.\n\n")
	if contactName != "" {
		b.WriteString("Nome na ficha do contacto (referência; não obrigatório em cada mensagem): ")
		b.WriteString(contactName)
		b.WriteString("\n\n")
	}

	total := utf8.RuneCountInString(b.String())
	for _, m := range msgs {
		line := formatMessageForLLM(&m)
		if strings.TrimSpace(line) == "" {
			continue
		}
		n := utf8.RuneCountInString(line) + 1
		if total+n > maxRunes {
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
		total += n
	}
	if b.Len() == 0 {
		return "", nil
	}
	return b.String(), nil
}

func formatMessageForLLM(m *model.Message) string {
	role := "Cliente"
	if m.Direction == "outbound" {
		role = "Assistente"
	}
	body := strings.TrimSpace(m.Body)
	if body == "" {
		mt := strings.TrimSpace(m.MessageType)
		if mt == "" {
			mt = "mensagem"
		}
		body = fmt.Sprintf("[%s]", mt)
	}
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\n", " ")
	if utf8.RuneCountInString(body) > 1500 {
		body = string([]rune(body)[:1500]) + "…"
	}
	return fmt.Sprintf("%s: %s", role, body)
}
