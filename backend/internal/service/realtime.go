package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// PublishInboxEvent notifica clientes (WebSocket / outro) via Redis Pub/Sub.
func PublishInboxEvent(rdb *redis.Client, workspaceID uuid.UUID, payload map[string]interface{}) {
	if rdb == nil || workspaceID == uuid.Nil {
		return
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	payload["workspace_id"] = workspaceID.String()
	payload["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	ch := "workspace:" + workspaceID.String() + ":events"
	_ = rdb.Publish(context.Background(), ch, b).Err()
}
