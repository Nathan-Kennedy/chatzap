package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"wa-saas/backend/internal/config"
)

const gravableFollowUpSystem = `És um assistente que produz um resumo executivo em português do Brasil para o cliente guardar ou copiar.
Extrai APENAS factos explícitos do texto (datas, horários, endereços, valores em reais, nomes de locais, prazos, próximos passos).
Formato obrigatório:
— começa com a linha "Resumo para registo:" (sem aspas)
— segue com linhas curtas, cada uma começando por "• " e um rótulo claro (ex.: Data/hora:, Local:, Valores:, Próximo passo:)
— sem saudações nem conversa; sem repetir frases longas do texto original
— se não houver nenhum dado concreto para registar, responde exactamente com uma única linha: SKIP`

// GravableFollowUpText gera texto profissional pós-áudio (LLM se possível; senão heurística).
func GravableFollowUpText(ctx context.Context, cfg *config.Config, originalReply string) (string, error) {
	s := strings.TrimSpace(originalReply)
	if s == "" {
		return "", fmt.Errorf("vazio")
	}
	if cfg != nil && strings.TrimSpace(cfg.GeminiAPIKey) != "" {
		g := NewGeminiClient(cfg.GeminiAPIKey, cfg.GeminiModel, gravableFollowUpSystem)
		out, err := g.Reply(ctx, "Texto da resposta do assistente (extrai só factos para o resumo):\n\n"+s)
		if err == nil {
			out = strings.TrimSpace(out)
			if out == "" || strings.EqualFold(out, "SKIP") || strings.HasPrefix(strings.ToLower(out), "skip") {
				return "", nil
			}
			return out, nil
		}
	}
	return heuristicGravableFollowUpPT(s), nil
}

var (
	reRS      = regexp.MustCompile(`(?i)r\$\s*[\d.,]+|[\d]+[\d.,]*\s*(reais|real)`)
	reDatePt  = regexp.MustCompile(`(?i)\b(\d{1,2}[/-]\d{1,2}([/-]\d{2,4})?|(segunda|ter[cç]a|quarta|quinta|sexta|s[aá]bado|domingo)|amanh[ãa]|hoje)\b`)
	reAddr    = regexp.MustCompile(`(?i)(avenida|av\.|rua|r\.|estrada|parque|bairro|n[º°]\s*\d+|número\s*\d+)`)
	reTime    = regexp.MustCompile(`(?i)\b(\d{1,2}\s*h(\d{2})?|às\s+\d{1,2}|manh[ãa]|tarde|noite)\b`)
	reOrç     = regexp.MustCompile(`(?i)(orçamento|orcamento|proposta|valor|agendamento|agendar|visita)`)
)

// heuristicGravableFollowUpPT fallback sem API.
func heuristicGravableFollowUpPT(s string) string {
	var b strings.Builder
	b.WriteString("Resumo para registo:\n")
	n := 0
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if utf8.RuneCountInString(line) < 12 {
			continue
		}
		if reOrç.MatchString(line) || reRS.MatchString(line) || reDatePt.MatchString(line) || reAddr.MatchString(line) || reTime.MatchString(line) {
			b.WriteString("• ")
			if len(line) > 220 {
				line = string([]rune(line)[:220]) + "…"
			}
			b.WriteString(line)
			b.WriteByte('\n')
			n++
		}
	}
	if n == 0 {
		// Uma linha genérica melhor que repetir tudo
		if len(s) > 400 {
			s = string([]rune(s)[:400]) + "…"
		}
		return "Resumo para registo:\n• Detalhes: " + s
	}
	return strings.TrimSpace(b.String())
}
