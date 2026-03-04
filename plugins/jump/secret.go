package jump

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

const jumpSecretPrefix = "enc:v1:"

func encryptJumpSecret(secretKey, plain string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", nil
	}
	// 兼容重复保存已加密值，避免二次加密。
	if strings.HasPrefix(plain, jumpSecretPrefix) {
		return plain, nil
	}
	block, err := aes.NewCipher(jumpSecretKey(secretKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	buf := append(nonce, ciphertext...)
	return jumpSecretPrefix + base64.StdEncoding.EncodeToString(buf), nil
}

func decryptJumpSecret(secretKey, val string) (string, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return "", nil
	}
	if !strings.HasPrefix(val, jumpSecretPrefix) {
		// 兼容历史明文数据。
		return val, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(val, jumpSecretPrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(jumpSecretKey(secretKey))
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

func jumpSecretKey(secretKey string) []byte {
	sum := sha256.Sum256([]byte("lazy-aiops-jump:" + strings.TrimSpace(secretKey)))
	return sum[:]
}
