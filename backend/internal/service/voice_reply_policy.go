package service

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// VoiceReplyShortMaxRunes — abaixo disto a auto-resposta envia só texto (mesmo com TTS ligado).
// Mensagens longas usam voz (TTS) quando o agente tem voz ativa.
const VoiceReplyShortMaxRunes = 240

// gravableInfoRE detecta conteúdo que o cliente costuma querer copiar (valores, datas, contactos).
var gravableInfoRE = regexp.MustCompile(`(?i)(orçamento|orcamento|agendamento|agendar|valores?|r\$\s*[\d.,]+|[\d]{1,2}/[\d]{1,2}(/[\d]{2,4})?|hor[áa]rio|prazo|proposta|contrato|pagamento|pix|boleto|endere[çc]o|conta banc|telefone|whatsapp|e-?mail|\b[\d]{2}\s*[\d]{4,5}[-\s.]?[\d]{4}\b|\bh[áa]\s+\d{1,2}h)`)

// ReplyLooksGravablePT indica se convém enviar texto por escrito a seguir ao áudio (dados para guardar).
func ReplyLooksGravablePT(s string) bool {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) < 16 {
		return false
	}
	return gravableInfoRE.MatchString(s)
}

// PreferVoiceForAutoReply devolve true quando a resposta é "longa" o suficiente para TTS.
func PreferVoiceForAutoReply(reply string) bool {
	return utf8.RuneCountInString(strings.TrimSpace(reply)) >= VoiceReplyShortMaxRunes
}
