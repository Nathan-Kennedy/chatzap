package service

import (
	"strings"
	"testing"
)

func TestSanitizeLLMTextForWhatsApp_boldishLabel(t *testing.T) {
	in := "*   *Projetos estruturais:* fazemos X."
	out := SanitizeLLMTextForWhatsApp(in)
	if strings.Contains(out, "*") {
		t.Fatalf("ainda tem asterisco: %q", out)
	}
	if !strings.Contains(out, "Projetos estruturais") {
		t.Fatalf("perdeu conteúdo: %q", out)
	}
}

func TestSanitizeLLMTextForWhatsApp_doubleStar(t *testing.T) {
	out := SanitizeLLMTextForWhatsApp("Use **atenção** aqui.")
	if strings.Contains(out, "*") {
		t.Fatalf("%q", out)
	}
	if !strings.Contains(out, "atenção") {
		t.Fatal(out)
	}
}

func TestSanitizeLLMTextForWhatsApp_limitsEmojis(t *testing.T) {
	in := "Olá 😊 tudo certo 😉 combinamos 🎉 amanhã ✨"
	out := SanitizeLLMTextForWhatsApp(in)
	if strings.Count(out, "😊")+strings.Count(out, "🎉")+strings.Count(out, "✨")+strings.Count(out, "😉") > 2 {
		t.Fatalf("esperava no máx. 2 emoji na saída: %q", out)
	}
	if !strings.Contains(out, "Olá") || !strings.Contains(out, "amanhã") {
		t.Fatalf("perdeu texto: %q", out)
	}
}
