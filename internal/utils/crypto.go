package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// EncryptField encrypts a string field using AES-256-GCM.
func EncryptField(plaintext string, keyHex string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := hexToBytes32(keyHex)
	if err != nil {
		return "", err
	}

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

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// DecryptField decrypts a base64url-encoded AES-256-GCM ciphertext.
func DecryptField(encoded string, keyHex string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	key, err := hexToBytes32(keyHex)
	if err != nil {
		return encoded, nil
	}

	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return encoded, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return encoded, nil
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return encoded, nil
	}

	if len(data) < gcm.NonceSize() {
		return encoded, nil
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return encoded, nil
	}

	return string(plaintext), nil
}

func hexToBytes32(keyHex string) ([]byte, error) {
	if len(keyHex) != 64 {
		return nil, errors.New("encryption key must be 64 hex chars (32 bytes)")
	}
	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		b, err := hexByte(keyHex[i*2], keyHex[i*2+1])
		if err != nil {
			return nil, errors.New("invalid hex in encryption key")
		}
		key[i] = b
	}
	return key, nil
}

func hexByte(hi, lo byte) (byte, error) {
	h, err := hexNibble(hi)
	if err != nil {
		return 0, err
	}
	l, err := hexNibble(lo)
	if err != nil {
		return 0, err
	}
	return (h << 4) | l, nil
}

func hexNibble(c byte) (byte, error) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', nil
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, nil
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, nil
	default:
		return 0, errors.New("invalid hex char")
	}
}
