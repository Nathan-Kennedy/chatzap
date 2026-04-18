package service

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestSplitReplyIntoMessageChunks_paragraphs(t *testing.T) {
	ch := SplitReplyIntoMessageChunks(
		"Este é o primeiro parágrafo com texto suficiente para não fundir com o seguinte.\n\nSegundo parágrafo separado por linha em branco.",
		200,
	)
	if len(ch) < 2 {
		t.Fatalf("esperava >=2 partes, got %v", ch)
	}
}

func TestSplitReplyIntoMessageChunks_twoLongBlocks(t *testing.T) {
	a := strings.Repeat("palavra ", 90)
	b := strings.Repeat("outro ", 90)
	ch := SplitReplyIntoMessageChunks(strings.TrimSpace(a)+"\n\n"+strings.TrimSpace(b), 120)
	if len(ch) < 2 {
		t.Fatalf("esperava vários chunks a partir de dois blocos, got %d: %q", len(ch), ch)
	}
	for _, c := range ch {
		if utf8.RuneCountInString(c) > 450 {
			t.Fatalf("chunk excessivo: %d runes", utf8.RuneCountInString(c))
		}
	}
}

func TestTypingDelayBeforeChunk_minimumAboutOneSecond(t *testing.T) {
	d := TypingDelayBeforeChunk("Oi", true)
	if d < 900*time.Millisecond {
		t.Fatalf("typing curto demais: %v", d)
	}
}

func TestPauseBetweenChunks_inRange(t *testing.T) {
	for range 20 {
		p := PauseBetweenChunks()
		if p < humanizePauseBetween {
			t.Fatalf("pause %v < min", p)
		}
		if p > humanizePauseBetween+humanizePauseJitterMax+time.Millisecond {
			t.Fatalf("pause %v > max esperado", p)
		}
	}
}
