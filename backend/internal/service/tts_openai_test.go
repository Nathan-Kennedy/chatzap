package service

import (
	"strings"
	"testing"

	"wa-saas/backend/internal/config"
)

func TestResolveStoredOpenAITTSVoice_omnivoiceDefaultClone(t *testing.T) {
	cfg := &config.Config{OmnivoiceDefaultTTSVoice: "clone:atendimento_br"}
	if got := ResolveStoredOpenAITTSVoice(TTSProviderOmnivoice, "", cfg); got != "clone:atendimento_br" {
		t.Fatalf("omnivoice vazio + env, got %q", got)
	}
	if got := ResolveStoredOpenAITTSVoice(TTSProviderOpenAI, "", cfg); got != "nova" {
		t.Fatalf("openai vazio ignora clone default, got %q", got)
	}
	if got := ResolveStoredOpenAITTSVoice(TTSProviderOmnivoice, "clone:other", cfg); got != "clone:other" {
		t.Fatalf("valor explícito, got %q", got)
	}
}

func TestResolveStoredOpenAITTSVoice_geminiDefaultKore(t *testing.T) {
	cfg := &config.Config{}
	if got := ResolveStoredOpenAITTSVoice(TTSProviderGemini, "", cfg); got != "Kore" {
		t.Fatalf("gemini vazio = Kore, got %q", got)
	}
	if got := ResolveStoredOpenAITTSVoice(TTSProviderGemini, "nova", cfg); got != "Kore" {
		t.Fatalf("gemini + nome OpenAI devia normalizar para Kore, got %q", got)
	}
	if got := ResolveStoredOpenAITTSVoice(TTSProviderGemini, "leda", cfg); got != "Leda" {
		t.Fatalf("gemini + leda, got %q", got)
	}
}

func TestResolveStoredOpenAITTSVoice_elevenlabsSanitizeOpenAIName(t *testing.T) {
	cfg := &config.Config{}
	want := "21m00Tcm4TlvDq8ikWAM"
	if got := ResolveStoredOpenAITTSVoice(TTSProviderElevenLabs, "nova", cfg); got != want {
		t.Fatalf("elevenlabs + nova (OpenAI) devia mapear para Rachel, got %q want %q", got, want)
	}
	if got := ResolveStoredOpenAITTSVoice(TTSProviderElevenLabs, "pf_dora", cfg); got != want {
		t.Fatalf("elevenlabs + pf_dora (Kokoro) devia mapear para Rachel, got %q", got)
	}
	if got := ResolveStoredOpenAITTSVoice(TTSProviderElevenLabs, "21m00Tcm4TlvDq8ikWAM", cfg); got != want {
		t.Fatalf("ID ElevenLabs válido mantém-se, got %q", got)
	}
}

func TestMapVoiceForOmnivoiceWithOpts_clonePassthrough(t *testing.T) {
	d := OmnivoiceAutoReplyDefaults()
	got := mapVoiceForOmnivoiceWithOpts("clone:atendimento_br", d)
	if got != "clone:atendimento_br" {
		t.Fatalf("clone não deve virar design, got %q", got)
	}
}

func TestMapVoiceForOmnivoiceWithOpts_naturalFemale(t *testing.T) {
	d := OmnivoiceAutoReplyDefaults()
	if !d.NaturalFemaleTimbre {
		t.Fatal("NaturalFemaleTimbre esperado")
	}
	v := mapVoiceForOmnivoiceWithOpts("nova", d)
	if !strings.HasPrefix(v, "design:") {
		t.Fatalf("nova com NaturalFemale devia usar design, got %q", v)
	}
	v2 := mapVoiceForOmnivoiceWithOpts("shimmer", d)
	if !strings.HasPrefix(v2, "design:") {
		t.Fatalf("shimmer com NaturalFemale devia usar design, got %q", v2)
	}
	v3 := mapVoiceForOmnivoiceWithOpts("nova", nil)
	if v3 != "auto" {
		t.Fatalf("nova sem opts = auto, got %q", v3)
	}
}

func TestVoiceReplyChunks_multipleSegments(t *testing.T) {
	long := strings.Repeat("Uma frase curta. ", 40)
	ch := voiceReplyChunks(long + "\n\n" + strings.Repeat("Outro bloco. ", 40))
	if len(ch) < 2 {
		t.Fatalf("esperava vários segmentos de voz, got %d", len(ch))
	}
}
