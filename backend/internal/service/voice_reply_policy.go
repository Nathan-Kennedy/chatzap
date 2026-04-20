package service

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// VoiceReplyShortMaxRunes — mensagens com pelo menos tantos caracteres usam voz (TTS) quando o agente tem voz ativa.
// Respostas mais curtas usam só texto, exceto se forem «operacionais» (ReplyLooksGravablePT).
const VoiceReplyShortMaxRunes = 240

// gravableInfoRE detecta orçamento, agendamento, valores, contactos — voz TTS + texto a seguir para copiar.
var gravableInfoRE = regexp.MustCompile(`(?i)(orçamento|orcamento|agendamento|agendar|marcar.{0,48}(visita|reuni[aã]o|hor[aá]|dia|hora)|visita\s+t[ée]cnica|reuni[aã]o|inspe[çc][aã]o|valores?|r\$\s*[\d.,]+|[\d]{1,2}/[\d]{1,2}(/[\d]{2,4})?|hor[áa]rio|prazo|proposta|contrato|pagamento|pix|boleto|endere[çc]o|conta banc|telefone|whatsapp|e-?mail|\b[\d]{2}\s*[\d]{4,5}[-\s.]?[\d]{4}\b|\bh[áa]\s+\d{1,2}h)`)

// ReplyLooksGravablePT indica se convém enviar texto por escrito a seguir ao áudio (dados para guardar).
func ReplyLooksGravablePT(s string) bool {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) < 16 {
		return false
	}
	return gravableInfoRE.MatchString(s)
}

// PreferVoiceForAutoReply: voz para mensagens longas OU para conteúdo operacional (orçamento, agendamento, valores…),
// mesmo curtas — alinhado com pedido de nota de voz nesses assuntos.
func PreferVoiceForAutoReply(reply string) bool {
	s := strings.TrimSpace(reply)
	if utf8.RuneCountInString(s) >= VoiceReplyShortMaxRunes {
		return true
	}
	return ReplyLooksGravablePT(s)
}
