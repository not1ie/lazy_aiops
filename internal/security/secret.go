package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const encryptedSecretPrefix = "enc:v1:"

func Encrypt(secretKey, scope, plain string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", nil
	}
	if strings.HasPrefix(plain, encryptedSecretPrefix) {
		return plain, nil
	}
	block, err := aes.NewCipher(deriveKey(secretKey, scope))
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
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	payload := append(nonce, ciphertext...)
	return encryptedSecretPrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func Decrypt(secretKey, scope, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, encryptedSecretPrefix) {
		return value, nil
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, encryptedSecretPrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(deriveKey(secretKey, scope))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted secret payload")
	}
	nonce := raw[:gcm.NonceSize()]
	ciphertext := raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func deriveKey(secretKey, scope string) []byte {
	sum := sha256.Sum256([]byte("lazy-aiops:" + strings.TrimSpace(scope) + ":" + strings.TrimSpace(secretKey)))
	return sum[:]
}
