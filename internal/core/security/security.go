package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 64 * 1024 // 64 MB
	argonIterations  = 3
	argonParallelism = 4
	argonKeyLength   = 32 // 256 bits for AES-256
	canaryText       = "anyzone:verified"
)

var (
	ErrInvalidPIN      = errors.New("PIN 码错误")
	ErrTooManyAttempts = errors.New("输错次数过多，请稍后再试")
	ErrPINNotConfigured = errors.New("尚未配置 PIN 码")
)

type SecurityManager struct {
	mu           sync.Mutex
	failedCount  int
	lockedUntil  time.Time
}

func NewSecurityManager() *SecurityManager {
	return &SecurityManager{}
}

// GenerateSalt 生成 16 字节随机盐
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// DeriveKey 使用 Argon2id 从 PIN 码派生 256 位密钥
func DeriveKey(pin string, salt []byte) []byte {
	return argon2.IDKey([]byte(pin), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
}

// Encrypt 使用 AES-256-GCM 加密明文
func Encrypt(key []byte, plaintext []byte) (string, error) {
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

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 使用 AES-256-GCM 解密密文
func Decrypt(key []byte, ciphertextBase64 string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("密文数据长度异常")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// GenerateVerifier 为新设置的 PIN 生成加密金丝雀验证串
func GenerateVerifier(key []byte) (string, error) {
	return Encrypt(key, []byte(canaryText))
}

// VerifyPIN 校验输入的 PIN 码，并包含防爆破锁定
func (m *SecurityManager) VerifyPIN(pin string, salt []byte, verifier string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否处于锁定期
	now := time.Now()
	if now.Before(m.lockedUntil) {
		remaining := int(m.lockedUntil.Sub(now).Seconds())
		return nil, fmt.Errorf("%w（剩余冷却时间：%d 秒）", ErrTooManyAttempts, remaining)
	}

	key := DeriveKey(pin, salt)
	plaintext, err := Decrypt(key, verifier)
	if err != nil || string(plaintext) != canaryText {
		m.failedCount++
		if m.failedCount >= 5 {
			m.lockedUntil = now.Add(5 * time.Minute)
		} else if m.failedCount >= 3 {
			m.lockedUntil = now.Add(30 * time.Second)
		}
		return nil, ErrInvalidPIN
	}

	// 验证成功，重置失败计数
	m.failedCount = 0
	m.lockedUntil = time.Time{}
	return key, nil
}
