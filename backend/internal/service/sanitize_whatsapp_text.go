package service

import (
	"regexp"
	"strings"

	"github.com/rivo/uniseg"
)

var (
	// "*   *texto" ou "* *" (lista + negrito Markdown quebrado)
	reBrokenAsterisks = regexp.MustCompile(`\*\s+\*`)
	reDoubleAsterisk  = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reSingleAsterisk  = regexp.MustCompile(`\*([^*\n]+)\*`)
	reLineListStar    = regexp.MustCompile(`(?m)^(\s*)\*\s+`)
	reMultiSpace      = regexp.MustCompile(`[ \t]{2,}`)
	reLonelyStarLine  = regexp.MustCompile(`(?m)^\s*\*\s*\n`)
	reStarAfterPunct  = regexp.MustCompile(`([:;.!?])\*+`)
	reStarsEOL        = regexp.MustCompile(`(?m)\*+\s*$`)
)

// sanitizeMaxEmojiGraphemes teto de «caracteres» emoji (clusters Unicode) por resposta após sanitizar.
const sanitizeMaxEmojiGraphemes = 2

// SanitizeLLMTextForWhatsApp remove padrões tipo Markdown (*negrito*, listas com *) comuns em respostas de LLM.
// O WhatsApp não formata isso como no Telegram/Markdown; o utilizador vê asteriscos soltos.
func SanitizeLLMTextForWhatsApp(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// Padrões quebrados: "*   *texto" ou "* *"
	for i := 0; i < 8; i++ {
		ns := reBrokenAsterisks.ReplaceAllString(s, " ")
		if ns == s {
			break
		}
		s = strings.TrimSpace(ns)
	}
	// **bloco**
	for i := 0; i < 16; i++ {
		ns := reDoubleAsterisk.ReplaceAllString(s, "$1")
		if ns == s {
			break
		}
		s = ns
	}
	// *frase* ou *rótulo:*
	for i := 0; i < 32; i++ {
		ns := reSingleAsterisk.ReplaceAllString(s, "$1")
		if ns == s {
			break
		}
		s = ns
	}
	// "*rótulo:*" por vezes fica "rótulo:*" (asterisco órfão após pontuação)
	s = reStarAfterPunct.ReplaceAllString(s, "$1")
	s = reStarsEOL.ReplaceAllString(s, "")
	s = reLonelyStarLine.ReplaceAllString(s, "\n")
	// Início de linha: "* item" vira "• item" (sem asterisco cru)
	s = reLineListStar.ReplaceAllString(s, "${1}• ")
	s = strings.ReplaceAll(s, "• •", "•")
	s = reMultiSpace.ReplaceAllString(s, " ")
	// Normalizar quebras com espaços estranhos
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(reMultiSpace.ReplaceAllString(lines[i], " "))
	}
	s = strings.Join(lines, "\n")
	s = strings.TrimSpace(s)
	return strings.TrimSpace(limitEmojiGraphemes(s, sanitizeMaxEmojiGraphemes))
}

// limitEmojiGraphemes mantém no máximo maxKeep clusters que parecem emoji (resto removido).
func limitEmojiGraphemes(s string, maxKeep int) string {
	if maxKeep < 0 || s == "" {
		return s
	}
	gr := uniseg.NewGraphemes(s)
	var b strings.Builder
	kept := 0
	for gr.Next() {
		cluster := gr.Str()
		if graphemeClusterIsEmoji(cluster) {
			if kept >= maxKeep {
				continue
			}
			kept++
		}
		b.WriteString(cluster)
	}
	return b.String()
}

func graphemeClusterIsEmoji(cluster string) bool {
	for _, r := range cluster {
		if r == 0x200D || r == 0xFE0F || (r >= 0x1F3FB && r <= 0x1F3FF) {
			continue
		}
		if isEmojiBaseRune(r) {
			return true
		}
	}
	return false
}

func isEmojiBaseRune(r rune) bool {
	switch {
	case r >= 0x1F1E6 && r <= 0x1F1FF:
		return true
	case r >= 0x1F300 && r <= 0x1FAFF:
		return true
	case r >= 0x2600 && r <= 0x26FF:
		return true
	case r >= 0x2700 && r <= 0x27BF:
		return true
	default:
		return false
	}
}
