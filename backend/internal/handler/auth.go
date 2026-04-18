package handler

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
	"wa-saas/backend/internal/service"
)

type registerBody struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	Name          string `json:"name"`
	WorkspaceName string `json:"workspace_name"`
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshBody struct {
	RefreshToken string `json:"refresh_token"`
}

func HandleRegister(log *zap.Logger, db *gorm.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body registerBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		body.Email = strings.TrimSpace(strings.ToLower(body.Email))
		body.Name = strings.TrimSpace(body.Name)
		body.WorkspaceName = strings.TrimSpace(body.WorkspaceName)
		if body.Email == "" || len(body.Password) < 8 {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "email e senha (mín. 8 caracteres) são obrigatórios", nil)
		}
		if body.WorkspaceName == "" {
			body.WorkspaceName = "Meu workspace"
		}

		var count int64
		if err := db.Model(&model.User{}).Where("email = ?", body.Email).Count(&count).Error; err != nil {
			log.Error("register count", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", "falha ao verificar email", nil)
		}
		if count > 0 {
			return JSONError(c, fiber.StatusConflict, "email_taken", "este email já está registado", nil)
		}

		hash, err := service.HashPassword(body.Password)
		if err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "hash_error", "falha ao processar senha", nil)
		}

		ws := &model.Workspace{Name: body.WorkspaceName}
		user := &model.User{Email: body.Email, PasswordHash: hash, Name: body.Name}

		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(ws).Error; err != nil {
				return err
			}
			if err := tx.Create(user).Error; err != nil {
				return err
			}
			m := &model.WorkspaceMember{
				WorkspaceID: ws.ID,
				UserID:      user.ID,
				Role:        "admin",
				CreatedAt:   time.Now(),
			}
			return tx.Create(m).Error
		})
		if err != nil {
			log.Error("register tx", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", "falha ao criar conta", nil)
		}

		return issueSession(c, log, db, cfg, user.ID, ws.ID, "admin", user.Email, user.Name)
	}
}

func HandleLogin(log *zap.Logger, db *gorm.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body loginBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		body.Email = strings.TrimSpace(strings.ToLower(body.Email))
		if body.Email == "" || body.Password == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "email e senha obrigatórios", nil)
		}

		var user model.User
		if err := db.Where("email = ?", body.Email).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return JSONError(c, fiber.StatusUnauthorized, "invalid_credentials", "email ou senha incorretos", nil)
			}
			log.Error("login lookup", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", "falha ao autenticar", nil)
		}
		if !service.CheckPassword(user.PasswordHash, body.Password) {
			return JSONError(c, fiber.StatusUnauthorized, "invalid_credentials", "email ou senha incorretos", nil)
		}

		var mem model.WorkspaceMember
		if err := db.Where("user_id = ?", user.ID).Order("created_at ASC").First(&mem).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return JSONError(c, fiber.StatusForbidden, "no_workspace", "utilizador sem workspace", nil)
			}
			log.Error("login member", zap.Error(err))
			return JSONError(c, fiber.StatusInternalServerError, "db_error", "falha ao autenticar", nil)
		}

		var ws model.Workspace
		if err := db.First(&ws, "id = ?", mem.WorkspaceID).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", "workspace não encontrado", nil)
		}

		return issueSession(c, log, db, cfg, user.ID, ws.ID, mem.Role, user.Email, user.Name)
	}
}

func issueSession(c *fiber.Ctx, log *zap.Logger, db *gorm.DB, cfg *config.Config, userID, workspaceID uuid.UUID, role, email, name string) error {
	access, err := service.IssueAccessToken(cfg, userID, workspaceID, role, email, name)
	if err != nil {
		log.Error("jwt access", zap.Error(err))
		return JSONError(c, fiber.StatusInternalServerError, "token_error", "falha ao emitir sessão", nil)
	}
	rawRefresh, hash, err := service.NewRefreshToken()
	if err != nil {
		return JSONError(c, fiber.StatusInternalServerError, "token_error", "falha ao emitir refresh", nil)
	}
	exp := time.Now().Add(time.Duration(cfg.JWTRefreshTTLDays) * 24 * time.Hour)
	rt := model.RefreshToken{
		UserID:      userID,
		WorkspaceID: workspaceID,
		TokenHash:   hash,
		ExpiresAt:   exp,
		CreatedAt:   time.Now(),
	}
	if err := db.Create(&rt).Error; err != nil {
		log.Error("refresh save", zap.Error(err))
		return JSONError(c, fiber.StatusInternalServerError, "db_error", "falha ao gravar sessão", nil)
	}

	var ws model.Workspace
	_ = db.First(&ws, "id = ?", workspaceID).Error

	return JSONSuccess(c, fiber.Map{
		"access_token":  access,
		"refresh_token": rawRefresh,
		"expires_in":    cfg.JWTAccessTTLMinutes * 60,
		"user": fiber.Map{
			"id":    userID.String(),
			"email": email,
			"name":  name,
			"role":  role,
		},
		"workspace_id":   workspaceID.String(),
		"workspace_name": ws.Name,
	})
}

func HandleRefresh(log *zap.Logger, db *gorm.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body refreshBody
		if err := c.BodyParser(&body); err != nil {
			return JSONError(c, fiber.StatusBadRequest, "invalid_body", "JSON inválido", nil)
		}
		if strings.TrimSpace(body.RefreshToken) == "" {
			return JSONError(c, fiber.StatusBadRequest, "validation_error", "refresh_token obrigatório", nil)
		}
		hash := service.HashRefreshToken(body.RefreshToken)
		var rt model.RefreshToken
		if err := db.Where("token_hash = ? AND expires_at > ?", hash, time.Now()).First(&rt).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return JSONError(c, fiber.StatusUnauthorized, "invalid_refresh", "refresh token inválido ou expirado", nil)
			}
			return JSONError(c, fiber.StatusInternalServerError, "db_error", "falha ao validar sessão", nil)
		}

		var user model.User
		if err := db.First(&user, "id = ?", rt.UserID).Error; err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "invalid_refresh", "utilizador inválido", nil)
		}
		var mem model.WorkspaceMember
		if err := db.Where("user_id = ? AND workspace_id = ?", user.ID, rt.WorkspaceID).First(&mem).Error; err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "invalid_refresh", "membro inválido", nil)
		}
		var ws model.Workspace
		if err := db.First(&ws, "id = ?", rt.WorkspaceID).Error; err != nil {
			return JSONError(c, fiber.StatusInternalServerError, "db_error", "workspace não encontrado", nil)
		}

		_ = db.Delete(&rt).Error

		return issueSession(c, log, db, cfg, user.ID, ws.ID, mem.Role, user.Email, user.Name)
	}
}

func HandleLogout(log *zap.Logger, db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body refreshBody
		_ = c.BodyParser(&body)
		if strings.TrimSpace(body.RefreshToken) != "" {
			hash := service.HashRefreshToken(body.RefreshToken)
			_ = db.Where("token_hash = ?", hash).Delete(&model.RefreshToken{}).Error
		}
		return JSONSuccess(c, fiber.Map{"ok": true})
	}
}

func HandleMe(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		uidStr, ok := c.Locals(middleware.LocalUserID).(string)
		if !ok || uidStr == "" {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "sessão inválida", nil)
		}
		widStr, ok := c.Locals(middleware.LocalWorkspaceID).(string)
		if !ok || widStr == "" {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "sessão inválida", nil)
		}
		role, _ := c.Locals(middleware.LocalRole).(string)
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "sessão inválida", nil)
		}
		wid, err := uuid.Parse(widStr)
		if err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "sessão inválida", nil)
		}

		var user model.User
		if err := db.First(&user, "id = ?", uid).Error; err != nil {
			return JSONError(c, fiber.StatusUnauthorized, "unauthorized", "utilizador não encontrado", nil)
		}
		var ws model.Workspace
		if err := db.First(&ws, "id = ?", wid).Error; err != nil {
			return JSONError(c, fiber.StatusNotFound, "not_found", "workspace não encontrado", nil)
		}

		return JSONSuccess(c, fiber.Map{
			"user": fiber.Map{
				"id":    user.ID.String(),
				"email": user.Email,
				"name":  user.Name,
				"role":  role,
			},
			"workspace_id":   ws.ID.String(),
			"workspace_name": ws.Name,
		})
	}
}
