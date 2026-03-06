// Copyright © 2024-2025 chouette.21.00@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// 暗号化キー（32バイト = AES-256）
// 本番環境では環境変数や設定ファイルから読み込むことを推奨
//
//	暗号化キー（32バイト = AES-256）を環境変数から64桁の16進数文字列として取得し、バイト列に変換する
var encryptionKey []byte

func init() {
	keyHex := os.Getenv("ENCRYPTION_KEY_HEX")
	if len(keyHex) != 64 {
		panic("ENCRYPTION_KEY_HEX must be a 64-character hexadecimal string")
	}
	encryptionKey = make([]byte, 32)
	for i := range 32 {
		var byteVal byte
		_, err := fmt.Sscanf(keyHex[i*2:i*2+2], "%02x", &byteVal)
		if err != nil {
			panic("Invalid hexadecimal character in ENCRYPTION_KEY_HEX")
		}
		encryptionKey[i] = byteVal
	}
}

// EncryptStr は文字列を暗号化してBase64エンコードされた文字列として返す
func EncryptStr(str string) (string, error) {
	if str == "" {
		return "", nil
	}

	block, err := aes.NewCipher(encryptionKey)
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

	ciphertext := gcm.Seal(nonce, nonce, []byte(str), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// DecryptStr は暗号化された文字列を復号化する
func DecryptStr(encryptedStr string) (string, error) {
	if encryptedStr == "" {
		return "", nil
	}

	ciphertext, err := base64.URLEncoding.DecodeString(encryptedStr)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// HashPassword はパスワードをハッシュ化する
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash はパスワードとハッシュを比較する
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
