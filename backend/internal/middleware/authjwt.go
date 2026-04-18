package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/service"
)

const (
	LocalUserID      = "auth_user_id"
	LocalWorkspaceID = "auth_workspace_id"
	LocalRole        = "auth_role"
	LocalEmail       = "auth_email"
	LocalName        = "auth_name"
)

// RequireAuth valida Bearer JWT de acesso.
func RequireAuth(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		h := c.Get("Authorization")
		if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			return jsonAuthErr(c, "token ausente ou inválido")
		}
		raw := strings.TrimSpace(h[7:])
		claims, err := service.ParseAccessToken(cfg, raw)
		if err != nil {
			return jsonAuthErr(c, "token inválido ou expirado")
		}
		if _, err := uuid.Parse(claims.UserID); err != nil {
			return jsonAuthErr(c, "token malformado")
		}
		if _, err := uuid.Parse(claims.WorkspaceID); err != nil {
			return jsonAuthErr(c, "token malformado")
		}
		c.Locals(LocalUserID, claims.UserID)
		c.Locals(LocalWorkspaceID, claims.WorkspaceID)
		c.Locals(LocalRole, claims.Role)
		c.Locals(LocalEmail, claims.Email)
		c.Locals(LocalName, claims.Name)
		return c.Next()
	}
}

func jsonAuthErr(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": fiber.Map{"code": "unauthorized", "message": message},
	})
}
