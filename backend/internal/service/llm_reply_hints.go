package service

import (
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

// ContinuationStyleHint texto curto a juntar ao pedido do utilizador para o modelo não repetir
// «Olá, [nome]» quando já houve cumprimento recente do assistente (mesmo que a última bolha não comece por «Olá»).
func ContinuationStyleHint(db *gorm.DB, conversationID uuid.UUID) string {
	var msgs []model.Message
	if err := db.Where("conversation_id = ? AND direction = ?", conversationID, "outbound").
		Order("created_at DESC").
		Limit(12).
		Find(&msgs).Error; err != nil || len(msgs) == 0 {
		return ""
	}
	for _, m := range msgs {
		if recentAssistantLineLooksLikeGreeting(m.Body) {
			return "\n[Estilo: já cumprimentaste o cliente há pouco nesta conversa. Nesta resposta não voltas a dizer «Olá» nem a repetir o nome ou apelido no início; responde direto ao pedido dele, sem nova saudação. Não inventes diminutivos do nome (ex.: «Nathanzinho») se o cliente não usou esse tratamento.]\n"
		}
	}
	return ""
}

func recentAssistantLineLooksLikeGreeting(body string) bool {
	first := firstLineOnly(strings.TrimSpace(body))
	if first == "" {
		return false
	}
	low := strings.ToLower(first)
	if strings.HasPrefix(low, "olá") || strings.HasPrefix(low, "ola,") || strings.HasPrefix(low, "ola ") {
		return true
	}
	if strings.HasPrefix(low, "oi,") || strings.HasPrefix(low, "oi ") || low == "oi" || low == "oi!" {
		return true
	}
	if strings.HasPrefix(low, "hey") || strings.HasPrefix(low, "e aí") || strings.HasPrefix(low, "e ai") {
		return true
	}
	if strings.HasPrefix(low, "bom dia") || strings.HasPrefix(low, "boa tarde") || strings.HasPrefix(low, "boa noite") {
		return true
	}
	return false
}

func firstLineOnly(s string) string {
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

const salutationNameLineMaxRunes = 90

// StripLeadingSalutationNameLine remove a primeira linha do tipo «Olá, Nome!» quando há texto útil na linha seguinte
// (fallback quando o modelo ignora a instrução de não repetir saudação).
func StripLeadingSalutationNameLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	idx := strings.IndexByte(s, '\n')
	if idx < 0 {
		return s
	}
	first := strings.TrimSpace(s[:idx])
	rest := strings.TrimSpace(s[idx+1:])
	if rest == "" {
		return s
	}
	if !isSalutationCommaNameLine(first) {
		return s
	}
	return rest
}

// palavras comuns após «Oi,» / «Olá,» que indicam frase de cortesia, não «nome + resto».
var salutationSecondWordBlacklist = map[string]struct{}{
	"tudo": {}, "como": {}, "posso": {}, "quer": {}, "qual": {}, "quando": {}, "onde": {},
	"preciso": {}, "favor": {}, "bem": {}, "vai": {}, "você": {}, "voce": {}, "vc": {},
	"está": {}, "esta": {}, "aí": {}, "ai": {},
}

func isSalutationCommaNameLine(line string) bool {
	if utf8.RuneCountInString(line) > salutationNameLineMaxRunes {
		return false
	}
	low := strings.ToLower(strings.TrimSpace(line))
	if !(strings.HasPrefix(low, "olá,") || strings.HasPrefix(low, "ola,") || strings.HasPrefix(low, "oi,")) {
		return false
	}
	fields := strings.Fields(line)
	if len(fields) < 2 || len(fields) > 6 {
		return false
	}
	w := strings.TrimRight(strings.ToLower(fields[1]), "!?.…")
	w = strings.Trim(w, "'’")
	if _, bad := salutationSecondWordBlacklist[w]; bad {
		return false
	}
	return true
}
