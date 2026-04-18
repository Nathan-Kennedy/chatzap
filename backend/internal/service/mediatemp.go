package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wa-saas/backend/internal/model"
)

// ErrMediaTooLarge ficheiro excede o limite configurado.
var ErrMediaTooLarge = errors.New("ficheiro excede o tamanho máximo permitido")

// NewMediaTempToken gera token opaco, grava conteúdo em disco e persiste a linha (TTL).
func NewMediaTempToken(db *gorm.DB, uploadDir string, ttl time.Duration, origFileName, mime string, src io.Reader, maxBytes int64) (plainToken string, _ *model.MediaTempToken, err error) {
	if err := os.MkdirAll(uploadDir, 0o750); err != nil {
		return "", nil, err
	}
	var rnd [16]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return "", nil, err
	}
	plainToken = hex.EncodeToString(rnd[:])
	safeName := sanitizeUploadFileName(origFileName)
	if safeName == "" {
		safeName = "file"
	}
	sub := uuid.New().String()
	dir := filepath.Join(uploadDir, sub)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", nil, err
	}
	fullPath := filepath.Join(dir, safeName)
	f, err := os.Create(fullPath)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	written, err := io.Copy(f, io.LimitReader(src, maxBytes+1))
	if err != nil {
		_ = os.Remove(fullPath)
		_ = os.Remove(dir)
		return "", nil, err
	}
	if written > maxBytes {
		_ = os.Remove(fullPath)
		_ = os.Remove(dir)
		return "", nil, ErrMediaTooLarge
	}
	now := time.Now().UTC()
	row := &model.MediaTempToken{
		Token:     plainToken,
		FilePath:  fullPath,
		MimeType:  strings.TrimSpace(mime),
		FileName:  strings.TrimSpace(origFileName),
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
	if err := db.Create(row).Error; err != nil {
		_ = os.Remove(fullPath)
		_ = os.Remove(dir)
		return "", nil, err
	}
	return plainToken, row, nil
}

func sanitizeUploadFileName(s string) string {
	s = filepath.Base(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "..", "")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// DeleteMediaTempToken remove ficheiro e linha.
func DeleteMediaTempToken(db *gorm.DB, row *model.MediaTempToken) {
	if row == nil {
		return
	}
	if row.FilePath != "" {
		_ = os.Remove(row.FilePath)
		dir := filepath.Dir(row.FilePath)
		_ = os.Remove(dir)
	}
	_ = db.Delete(&model.MediaTempToken{}, "id = ?", row.ID).Error
}

// PurgeExpiredMediaTokens apaga tokens expirados e ficheiros órfãos.
func PurgeExpiredMediaTokens(db *gorm.DB, log *zap.Logger) {
	now := time.Now().UTC()
	var rows []model.MediaTempToken
	if err := db.Where("expires_at < ?", now).Find(&rows).Error; err != nil {
		if log != nil {
			log.Debug("purge media tokens", zap.Error(err))
		}
		return
	}
	for i := range rows {
		DeleteMediaTempToken(db, &rows[i])
	}
}
