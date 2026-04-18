package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"wa-saas/backend/internal/pkg/securestring"
)

const headerInternalKey = "X-Internal-API-Key"

// InternalAPIKey protege rotas operacionais até existir JWT + RBAC.
func InternalAPIKey(expected string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		got := strings.TrimSpace(c.Get(headerInternalKey))
		if got == "" || !securestring.Equal(got, expected) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"code": "unauthorized", "message": "chave interna inválida ou ausente"},
			})
		}
		return c.Next()
	}
}
