package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAutoReplyDebouncer_batchesWithinWindow(t *testing.T) {
	d := NewAutoReplyDebouncer(80 * time.Millisecond)
	cid := uuid.New()
	var mu sync.Mutex
	var batches [][]AutoReplyQueueItem
	flush := func(ctx context.Context, batch []AutoReplyQueueItem) error {
		mu.Lock()
		batches = append(batches, append([]AutoReplyQueueItem(nil), batch...))
		mu.Unlock()
		return nil
	}
	d.Schedule(cid, AutoReplyQueueItem{MessageID: uuid.New(), Text: "a"}, flush)
	time.Sleep(30 * time.Millisecond)
	d.Schedule(cid, AutoReplyQueueItem{MessageID: uuid.New(), Text: "b"}, flush)
	time.Sleep(120 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	if len(batches[0]) != 2 {
		t.Fatalf("expected 2 items in batch, got %d", len(batches[0]))
	}
	if batches[0][0].Text != "a" || batches[0][1].Text != "b" {
		t.Fatalf("unexpected order: %#v", batches[0])
	}
}

func TestAutoReplyDebouncer_zeroDelayImmediate(t *testing.T) {
	d := NewAutoReplyDebouncer(0)
	cid := uuid.New()
	var n int
	var mu sync.Mutex
	flush := func(ctx context.Context, batch []AutoReplyQueueItem) error {
		mu.Lock()
		n++
		mu.Unlock()
		return nil
	}
	d.Schedule(cid, AutoReplyQueueItem{Text: "x"}, flush)
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if n != 1 {
		t.Fatalf("expected immediate flush, n=%d", n)
	}
}
