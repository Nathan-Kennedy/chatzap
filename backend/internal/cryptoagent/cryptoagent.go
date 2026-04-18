package cryptoagent

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// DeriveKey converte uma passphrase em chave AES-256 (32 bytes).
func DeriveKey(passphrase string) []byte {
	sum := sha256.Sum256([]byte(passphrase))
	return sum[:]
}

// Encrypt encripta texto em claro com AES-GCM; saída base64 (nonce || ciphertext).
func Encrypt(plaintext string, passphrase string) (string, error) {
	if passphrase == "" {
		return "", fmt.Errorf("APP_ENCRYPTION_KEY não configurada")
	}
	key := DeriveKey(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverte Encrypt.
func Decrypt(encoded string, passphrase string) (string, error) {
	if passphrase == "" {
		return "", fmt.Errorf("APP_ENCRYPTION_KEY não configurada")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("ciphertext inválido: %w", err)
	}
	key := DeriveKey(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return "", fmt.Errorf("ciphertext demasiado curto")
	}
	nonce, ct := data[:ns], data[ns:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("falha ao desencriptar: %w", err)
	}
	return string(plain), nil
}
