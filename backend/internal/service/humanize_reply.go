package service

import (
	"math/rand"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	humanizeMaxChunkRunes   = 380
	humanizeMinMergeRunes   = 12
	humanizeTypingMinFirst  = 1 * time.Second
	humanizeTypingMinLater  = 1 * time.Second
	humanizeTypingPerRune   = 28 * time.Millisecond
	humanizeTypingCapFirst  = 10 * time.Second
	humanizeTypingCapLater  = 6 * time.Second
	humanizePauseBetween    = 450 * time.Millisecond
	humanizePauseJitterMax  = 350 * time.Millisecond
)

// SplitReplyIntoMessageChunks divide a resposta do LLM em bolhas curtas (parágrafos, frases, limite de runes).
func SplitReplyIntoMessageChunks(reply string, maxRunes int) []string {
	if maxRunes <= 40 {
		maxRunes = humanizeMaxChunkRunes
	}
	s := strings.TrimSpace(reply)
	if s == "" {
		return nil
	}
	var out []string
	for _, block := range strings.Split(s, "\n\n") {
		b := strings.TrimSpace(block)
		if b == "" {
			continue
		}
		out = append(out, splitOversizedBlock(b, maxRunes)...)
	}
	return mergeTinyChunks(out, humanizeMinMergeRunes)
}

func splitOversizedBlock(block string, maxRunes int) []string {
	if utf8.RuneCountInString(block) <= maxRunes {
		return []string{block}
	}
	var pieces []string
	start := 0
	for start < len(block) {
		rest := block[start:]
		if utf8.RuneCountInString(rest) <= maxRunes {
			if t := strings.TrimSpace(rest); t != "" {
				pieces = append(pieces, t)
			}
			break
		}
		cut := cutAtSentenceOrSpace(rest, maxRunes)
		if cut == 0 {
			cut = hardCutRunes(rest, maxRunes)
		}
		chunk := strings.TrimSpace(block[start : start+cut])
		if chunk != "" {
			pieces = append(pieces, chunk)
		}
		start += cut
		for start < len(block) && (block[start] == ' ' || block[start] == '\n') {
			start++
		}
	}
	return pieces
}

func cutAtSentenceOrSpace(s string, maxRunes int) int {
	if maxRunes <= 0 || s == "" {
		return 0
	}
	runes := 0
	lastSpace := -1
	for i := 0; i < len(s); {
		r, w := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && w == 1 {
			i++
			continue
		}
		if r == ' ' || r == '\n' {
			lastSpace = i + w
		}
		if r == '.' || r == '!' || r == '?' {
			end := i + w
			for end < len(s) && s[end] == ' ' {
				end++
			}
			if utf8.RuneCountInString(s[:end]) <= maxRunes {
				return end
			}
		}
		runes++
		if runes > maxRunes {
			if lastSpace > 0 {
				return lastSpace
			}
			return i
		}
		i += w
	}
	return len(s)
}

func hardCutRunes(s string, maxRunes int) int {
	if maxRunes <= 0 {
		return 0
	}
	runes := 0
	i := 0
	for i < len(s) {
		_, w := utf8.DecodeRuneInString(s[i:])
		runes++
		i += w
		if runes >= maxRunes {
			break
		}
	}
	return i
}

func mergeTinyChunks(chunks []string, minRunes int) []string {
	if len(chunks) == 0 {
		return nil
	}
	var merged []string
	var buf strings.Builder
	for _, c := range chunks {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if buf.Len() == 0 {
			buf.WriteString(c)
			continue
		}
		if utf8.RuneCountInString(buf.String()) < minRunes {
			buf.WriteString(" ")
			buf.WriteString(c)
			continue
		}
		merged = append(merged, buf.String())
		buf.Reset()
		buf.WriteString(c)
	}
	if buf.Len() > 0 {
		merged = append(merged, buf.String())
	}
	return merged
}

// TypingDelayBeforeChunk duração do indicador "digitando…" antes de enviar este segmento (mín. ~1s por bolha).
func TypingDelayBeforeChunk(chunk string, isFirst bool) time.Duration {
	n := utf8.RuneCountInString(chunk)
	if n < 1 {
		n = 1
	}
	per := time.Duration(n) * humanizeTypingPerRune
	var d time.Duration
	if isFirst {
		d = humanizeTypingMinFirst + per
		if d > humanizeTypingCapFirst {
			d = humanizeTypingCapFirst
		}
	} else {
		d = humanizeTypingMinLater + per
		if d > humanizeTypingCapLater {
			d = humanizeTypingCapLater
		}
	}
	return d
}

// PauseBetweenChunks pausa entre bolhas consecutivas (parece tempo entre mensagens humanas).
func PauseBetweenChunks() time.Duration {
	if humanizePauseJitterMax <= 0 {
		return humanizePauseBetween
	}
	return humanizePauseBetween + time.Duration(rand.Int63n(int64(humanizePauseJitterMax)))
}
