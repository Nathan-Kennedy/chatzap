package service

import (
	"strings"
	"testing"
)

func TestPreferVoiceForAutoReply(t *testing.T) {
	short := strings.Repeat("a", VoiceReplyShortMaxRunes-1)
	if PreferVoiceForAutoReply(short) {
		t.Fatal("curto casual demais devia ser só texto")
	}
	long := strings.Repeat("b", VoiceReplyShortMaxRunes)
	if !PreferVoiceForAutoReply(long) {
		t.Fatal("longo devia preferir voz")
	}
	operacional := "Podemos agendar a visita técnica para terça. O orçamento base é R$ 800."
	if !PreferVoiceForAutoReply(operacional) {
		t.Fatal("curto mas operacional devia preferir voz")
	}
}

func TestReplyLooksGravablePT(t *testing.T) {
	if ReplyLooksGravablePT("só um oi") {
		t.Fatal("demasiado curto")
	}
	if !ReplyLooksGravablePT("O orçamento fica em R$ 1.200,00 e o prazo é 15 dias.") {
		t.Fatal("devia detectar valores/prazo")
	}
	if !ReplyLooksGravablePT("Agendamento para segunda às 14h no escritório.") {
		t.Fatal("devia detectar agendamento")
	}
}
