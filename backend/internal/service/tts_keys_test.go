package service

import "testing"

func TestNormalizeTTSProvider(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", TTSProviderNone},
		{"  ", TTSProviderNone},
		{"none", TTSProviderNone},
		{"NONE", TTSProviderNone},
		{"openai_tts", TTSProviderOpenAI},
		{"omnivoice", TTSProviderOmnivoice},
		{"elevenlabs", TTSProviderElevenLabs},
		{"kokoro", TTSProviderKokoro},
		{"unknown", TTSProviderNone},
	}
	for _, c := range cases {
		if got := NormalizeTTSProvider(c.in); got != c.want {
			t.Errorf("NormalizeTTSProvider(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
