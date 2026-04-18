package handler

import (
	"context"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/service"
)

const wsTextMessage = 1

// WebSocketRoute GET /ws?token= — valida JWT e subscreve Redis do workspace.
func WebSocketRoute(cfg *config.Config, log *zap.Logger, rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			tok := c.Query("token")
			claims, err := service.ParseAccessToken(cfg, tok)
			if err != nil {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
			wid, err := uuid.Parse(claims.WorkspaceID)
			if err != nil {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
			h := websocket.New(func(conn *websocket.Conn) {
				subscribeRedisWS(conn, rdb, wid, log)
			})
			return h(c)
		}
		return fiber.ErrUpgradeRequired
	}
}

func subscribeRedisWS(c *websocket.Conn, rdb *redis.Client, wid uuid.UUID, log *zap.Logger) {
	if rdb == nil {
		_ = c.Close()
		return
	}
	ctx := context.Background()
	chName := "workspace:" + wid.String() + ":events"
	sub := rdb.Subscribe(ctx, chName)
	defer func() { _ = sub.Close() }()

	for {
		msg, err := sub.ReceiveMessage(ctx)
		if err != nil {
			log.Debug("ws redis recv", zap.Error(err))
			return
		}
		if err := c.WriteMessage(wsTextMessage, []byte(msg.Payload)); err != nil {
			return
		}
	}
}
