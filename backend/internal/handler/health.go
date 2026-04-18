package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthDeps dependências para checks.
type HealthDeps struct {
	DB    *gorm.DB
	Redis *redis.Client
}

// HandleHealth readiness básico (Postgres + Redis).
func HandleHealth(deps HealthDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		pgOK := true
		if sqlDB, err := deps.DB.DB(); err != nil {
			pgOK = false
		} else if err := sqlDB.PingContext(ctx); err != nil {
			pgOK = false
		}

		redisOK := deps.Redis.Ping(ctx).Err() == nil

		status := fiber.StatusOK
		if !pgOK || !redisOK {
			status = fiber.StatusServiceUnavailable
		}

		return c.Status(status).JSON(fiber.Map{
			"data": fiber.Map{
				"status": map[bool]string{true: "ok", false: "degraded"}[pgOK && redisOK],
				"checks": fiber.Map{
					"postgres": pgOK,
					"redis":    redisOK,
				},
			},
		})
	}
}
