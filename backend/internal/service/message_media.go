package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// extFromMime sugere extensão para ficheiros persistidos (WhatsApp / Evolution).
func extFromMime(mime string) string {
	mime = strings.TrimSpace(strings.ToLower(strings.Split(mime, ";")[0]))
	switch mime {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "audio/ogg":
		return ".ogg"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/mp4", "audio/aac":
		return ".m4a"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

// WriteMessageMediaBytes grava mídia recebida/decodificada para MEDIA_PERSISTENT_DIR (<messageID><ext>).
func WriteMessageMediaBytes(persistDir string, messageID uuid.UUID, data []byte, origFileName, mime string) (relKey string, _ error) {
	if len(data) == 0 {
		return "", fmt.Errorf("dados vazios")
	}
	persistDir = strings.TrimSpace(persistDir)
	if persistDir == "" {
		return "", fmt.Errorf("persist dir vazio")
	}
	if err := os.MkdirAll(persistDir, 0o750); err != nil {
		return "", err
	}
	ext := filepath.Ext(strings.TrimSpace(origFileName))
	if ext == "" {
		ext = extFromMime(mime)
	}
	if ext == "" {
		ext = ".bin"
	}
	relKey = messageID.String() + ext
	dest := filepath.Join(persistDir, relKey)
	if err := os.WriteFile(dest, data, 0o640); err != nil {
		return "", err
	}
	return relKey, nil
}

// CopyMessageMediaToPersistent copia o ficheiro temporário (upload Evolution) para MEDIA_PERSISTENT_DIR.
// Grava apenas o nome relativo seguro: <messageID><ext>.
func CopyMessageMediaToPersistent(persistDir, tempFilePath string, messageID uuid.UUID, origFileName string) (relKey string, _ error) {
	tempFilePath = strings.TrimSpace(tempFilePath)
	if tempFilePath == "" {
		return "", fmt.Errorf("temp path vazio")
	}
	persistDir = strings.TrimSpace(persistDir)
	if persistDir == "" {
		return "", fmt.Errorf("persist dir vazio")
	}
	if err := os.MkdirAll(persistDir, 0o750); err != nil {
		return "", err
	}
	ext := filepath.Ext(strings.TrimSpace(origFileName))
	if ext == "" {
		ext = ".bin"
	}
	relKey = messageID.String() + ext
	dest := filepath.Join(persistDir, relKey)
	src, err := os.Open(tempFilePath)
	if err != nil {
		return "", err
	}
	defer src.Close()
	out, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	return relKey, nil
}

// ResolvePersistentMediaPath devolve caminho absoluto do ficheiro se relKey estiver contido em persistDir.
func ResolvePersistentMediaPath(persistDir, relKey string) (abs string, _ error) {
	persistDir = strings.TrimSpace(persistDir)
	relKey = strings.TrimSpace(relKey)
	if persistDir == "" || relKey == "" {
		return "", fmt.Errorf("path inválido")
	}
	cleanRel := filepath.Clean(relKey)
	if strings.HasPrefix(cleanRel, "..") {
		return "", fmt.Errorf("path inválido")
	}
	absBase, err := filepath.Abs(persistDir)
	if err != nil {
		return "", err
	}
	full := filepath.Join(absBase, cleanRel)
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absBase, absFull)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path fora do diretório")
	}
	return absFull, nil
}
