package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AutoReplyQueueItem é um snapshot de uma mensagem inbound a incluir no lote da auto-resposta.
type AutoReplyQueueItem struct {
	MessageID          uuid.UUID
	Text               string
	WaMediaMessageJSON string
	MessageType        string
	KeyID              string
}

// AutoReplyDebouncer acumula mensagens por conversa e só dispara após Wait sem novas mensagens
// (debounce). Cada nova mensagem reinicia o temporizador. Em processo único; com várias réplicas
// da API o agrupamento pode não ser fiável sem estado partilhado (Redis).
type AutoReplyDebouncer struct {
	mu       sync.Mutex
	sessions map[uuid.UUID]*autoReplyDebounceSession
	Wait     time.Duration
}

type autoReplyDebounceSession struct {
	items []AutoReplyQueueItem
	timer *time.Timer
}

// NewAutoReplyDebouncer wait 0 = sem pausa (uma mensagem por flush imediato).
func NewAutoReplyDebouncer(wait time.Duration) *AutoReplyDebouncer {
	return &AutoReplyDebouncer{
		sessions: make(map[uuid.UUID]*autoReplyDebounceSession),
		Wait:     wait,
	}
}

// Schedule adiciona item ao lote da conversa e agenda flush após Wait (ou reinicia o timer).
// O flush corre noutra goroutine; não bloqueia o webhook.
func (d *AutoReplyDebouncer) Schedule(
	conversationID uuid.UUID,
	item AutoReplyQueueItem,
	flush func(ctx context.Context, batch []AutoReplyQueueItem) error,
) {
	if d == nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()
			_ = flush(ctx, []AutoReplyQueueItem{item})
		}()
		return
	}
	if d.Wait <= 0 {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()
			_ = flush(ctx, []AutoReplyQueueItem{item})
		}()
		return
	}

	d.mu.Lock()
	sess, ok := d.sessions[conversationID]
	if !ok {
		sess = &autoReplyDebounceSession{}
		d.sessions[conversationID] = sess
	}
	if sess.timer != nil {
		sess.timer.Stop()
	}
	sess.items = append(sess.items, item)
	sess.timer = time.AfterFunc(d.Wait, func() {
		d.mu.Lock()
		batch := append([]AutoReplyQueueItem(nil), sess.items...)
		delete(d.sessions, conversationID)
		d.mu.Unlock()
		if len(batch) == 0 {
			return
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()
			_ = flush(ctx, batch)
		}()
	})
	d.mu.Unlock()
}
