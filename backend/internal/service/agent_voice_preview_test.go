package service

import "testing"

func TestAgentVoicePreviewPhrase(t *testing.T) {
	p := AgentVoicePreviewPhrase("Maria")
	if p == "" || len(p) < 20 {
		t.Fatalf("phrase too short: %q", p)
	}
	p2 := AgentVoicePreviewPhrase("")
	if p2 == "" {
		t.Fatal("empty name should fallback")
	}
}

func TestVoicePreviewNeedsRegenerate(t *testing.T) {
	if !VoicePreviewNeedsRegenerate(map[string]interface{}{"name": "x"}) {
		t.Fatal("name should trigger")
	}
	if VoicePreviewNeedsRegenerate(map[string]interface{}{"description": "x"}) {
		t.Fatal("description should not trigger")
	}
}
