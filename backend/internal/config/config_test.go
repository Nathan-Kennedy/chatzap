package config

import (
	"os"
	"testing"
)

// Evita que um backend/.env local preencha variáveis durante os testes.
func isolateEnv(t *testing.T) func() {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() { _ = os.Chdir(wd) }
}

func TestLoad_autoReply_allowsMissingGlobalGeminiKey(t *testing.T) {
	defer isolateEnv(t)()
	clearBackendEnv(t)
	mustSet(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	mustSet(t, "EVOLUTION_BASE_URL", "http://127.0.0.1:8081")
	mustSet(t, "EVOLUTION_API_KEY", "ev-key")
	mustSet(t, "INTERNAL_API_KEY", "int-key")
	mustSet(t, "JWT_SECRET", "01234567890123456789012345678901")
	mustSet(t, "EVOLUTION_WEBHOOK_API_KEY", "wh-key")
	mustSet(t, "AUTO_REPLY_ENABLED", "true")
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("LLM_PROVIDER")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.AutoReplyEnabled || c.LLMProvider != "gemini" {
		t.Fatalf("auto=%v provider=%q", c.AutoReplyEnabled, c.LLMProvider)
	}
}

func TestLoad_autoReply_allowsMissingGlobalOpenAIKey(t *testing.T) {
	defer isolateEnv(t)()
	clearBackendEnv(t)
	mustSet(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	mustSet(t, "EVOLUTION_BASE_URL", "http://127.0.0.1:8081")
	mustSet(t, "EVOLUTION_API_KEY", "ev-key")
	mustSet(t, "INTERNAL_API_KEY", "int-key")
	mustSet(t, "JWT_SECRET", "01234567890123456789012345678901")
	mustSet(t, "EVOLUTION_WEBHOOK_API_KEY", "wh-key")
	mustSet(t, "AUTO_REPLY_ENABLED", "true")
	mustSet(t, "LLM_PROVIDER", "openai")
	mustSet(t, "GEMINI_API_KEY", "g-key")
	os.Unsetenv("OPENAI_API_KEY")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.AutoReplyEnabled || c.LLMProvider != "openai" {
		t.Fatalf("auto=%v provider=%q", c.AutoReplyEnabled, c.LLMProvider)
	}
}

func TestLoad_autoReply_geminiOK(t *testing.T) {
	defer isolateEnv(t)()
	clearBackendEnv(t)
	mustSet(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	mustSet(t, "EVOLUTION_BASE_URL", "http://127.0.0.1:8081")
	mustSet(t, "EVOLUTION_API_KEY", "ev-key")
	mustSet(t, "INTERNAL_API_KEY", "int-key")
	mustSet(t, "JWT_SECRET", "01234567890123456789012345678901")
	mustSet(t, "EVOLUTION_WEBHOOK_API_KEY", "wh-key")
	mustSet(t, "AUTO_REPLY_ENABLED", "true")
	mustSet(t, "GEMINI_API_KEY", "fake-gemini")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.LLMProvider != "gemini" {
		t.Fatalf("provider %q", c.LLMProvider)
	}
}

func TestLoad_webhookDisabledLLMKeysOptional(t *testing.T) {
	defer isolateEnv(t)()
	clearBackendEnv(t)
	mustSet(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	mustSet(t, "EVOLUTION_BASE_URL", "http://127.0.0.1:8081")
	mustSet(t, "EVOLUTION_API_KEY", "ev-key")
	mustSet(t, "INTERNAL_API_KEY", "int-key")
	mustSet(t, "JWT_SECRET", "01234567890123456789012345678901")
	mustSet(t, "EVOLUTION_WEBHOOK_API_KEY", "wh-key")
	mustSet(t, "AUTO_REPLY_ENABLED", "false")
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.AutoReplyEnabled {
		t.Fatal("auto reply should be off")
	}
}

func clearBackendEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"ENV", "DATABASE_URL", "REDIS_URL", "EVOLUTION_BASE_URL", "EVOLUTION_API_KEY",
		"EVOLUTION_INSTANCE_NAME", "WEBHOOK_SHARED_SECRET", "EVOLUTION_WEBHOOK_API_KEY",
		"INSECURE_SKIP_WEBHOOK_AUTH", "INTERNAL_API_KEY", "OPENAI_API_KEY", "OPENAI_MODEL",
		"GEMINI_API_KEY", "GEMINI_MODEL", "LLM_PROVIDER", "LLM_SYSTEM_PROMPT", "OPENAI_SYSTEM_PROMPT",
		"AUTO_REPLY_ENABLED", "ALLOWED_INSTANCE_IDS", "CORS_ALLOW_ORIGINS", "WHATSAPP_PROVIDER",
		"JWT_SECRET", "JWT_ACCESS_TTL_MINUTES", "JWT_REFRESH_TTL_DAYS",
		"HTTP_PORT", "PUBLIC_WEBHOOK_BASE_URL",
	}
	for _, k := range keys {
		_ = os.Unsetenv(k)
	}
}

func mustSet(t *testing.T, k, v string) {
	t.Helper()
	t.Setenv(k, v)
}

func TestLoad_noneWithoutEvolutionURLs(t *testing.T) {
	defer isolateEnv(t)()
	clearBackendEnv(t)
	mustSet(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	mustSet(t, "INTERNAL_API_KEY", "int-key")
	mustSet(t, "JWT_SECRET", "01234567890123456789012345678901")
	mustSet(t, "WHATSAPP_PROVIDER", "none")
	mustSet(t, "EVOLUTION_WEBHOOK_API_KEY", "wh-key")
	mustSet(t, "AUTO_REPLY_ENABLED", "false")
	os.Unsetenv("EVOLUTION_BASE_URL")
	os.Unsetenv("EVOLUTION_API_KEY")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.WhatsAppProvider != "none" {
		t.Fatalf("provider %q", c.WhatsAppProvider)
	}
}
