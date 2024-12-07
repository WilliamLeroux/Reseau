package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

func GenerateKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	key := hex.EncodeToString(bytes)
	return key, nil
}

func Encrypt(message string, encryptionKey string) (string, error) {
	key, err := hex.DecodeString(encryptionKey)
	if err != nil {

		return "", err
	}
	text := []byte(message)

	block, err := aes.NewCipher(key)
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

	cipherText := gcm.Seal(nonce, nonce, text, nil)
	return fmt.Sprintf("%x", cipherText), nil
}

func Decrypt(encryptedMessage string, decryptionKey string) (string, error) {
	key, err := hex.DecodeString(decryptionKey)
	if err != nil {
		return "", err
	}
	enc, err := hex.DecodeString(encryptedMessage)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", nil
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()

	nonce, cipherText := enc[:nonceSize], enc[nonceSize:]

	text, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}
	return string(text), nil
}
