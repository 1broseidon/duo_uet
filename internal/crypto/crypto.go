package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptedFieldPrefix identifies encrypted fields in YAML
const EncryptedFieldPrefix = "ENC[AES256_GCM,"

var encryptedFieldRegex = regexp.MustCompile(`ENC\[AES256_GCM,([^,]+),([^\]]+)\]`)

// CryptoManager handles encryption/decryption of sensitive configuration fields
type CryptoManager struct {
	masterKey []byte
}

// NewCryptoManager creates a new crypto manager with a master key
// The master key can come from:
// 1. Environment variable: UET_MASTER_KEY
// 2. Key file: .uet_key (in app directory)
// 3. Generated on first run and saved to .uet_key
func NewCryptoManager() (*CryptoManager, error) {
	// Try environment variable first
	if key := os.Getenv("UET_MASTER_KEY"); key != "" {
		derivedKey := deriveKey([]byte(key), []byte("uet-salt"))
		return &CryptoManager{masterKey: derivedKey}, nil
	}

	// Try key file
	keyPath := ".uet_key"
	if data, err := os.ReadFile(keyPath); err == nil {
		derivedKey := deriveKey(data, []byte("uet-salt"))
		return &CryptoManager{masterKey: derivedKey}, nil
	}

	// Generate new key and save
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Save to file with restricted permissions
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, fmt.Errorf("failed to save key file: %w", err)
	}

	derivedKey := deriveKey(key, []byte("uet-salt"))
	return &CryptoManager{masterKey: derivedKey}, nil
}

// NewCryptoManagerWithKey creates a crypto manager with a specific key (for testing)
func NewCryptoManagerWithKey(password string) *CryptoManager {
	derivedKey := deriveKey([]byte(password), []byte("uet-salt"))
	return &CryptoManager{masterKey: derivedKey}
}

// deriveKey derives a 32-byte encryption key from a password using PBKDF2
func deriveKey(password, salt []byte) []byte {
	return pbkdf2.Key(password, salt, 100000, 32, sha256.New)
}

// Encrypt encrypts plaintext and returns a formatted encrypted string
// Format: ENC[AES256_GCM,<nonce>,<ciphertext>]
func (cm *CryptoManager) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(cm.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Format: ENC[AES256_GCM,base64(nonce),base64(ciphertext)]
	nonceB64 := base64.StdEncoding.EncodeToString(nonce)
	ciphertextB64 := base64.StdEncoding.EncodeToString(ciphertext)

	return fmt.Sprintf("ENC[AES256_GCM,%s,%s]", nonceB64, ciphertextB64), nil
}

// Decrypt decrypts an encrypted field
// Returns the original string if not encrypted
func (cm *CryptoManager) Decrypt(encryptedField string) (string, error) {
	// If not encrypted, return as-is
	if !strings.HasPrefix(encryptedField, EncryptedFieldPrefix) {
		return encryptedField, nil
	}

	// Parse encrypted field: ENC[AES256_GCM,<nonce>,<ciphertext>]
	matches := encryptedFieldRegex.FindStringSubmatch(encryptedField)
	if len(matches) != 3 {
		return "", fmt.Errorf("invalid encrypted field format")
	}

	nonceB64 := matches[1]
	ciphertextB64 := matches[2]

	// Decode base64
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Decrypt
	block, err := aes.NewCipher(cm.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a field is encrypted
func IsEncrypted(field string) bool {
	return strings.HasPrefix(field, EncryptedFieldPrefix)
}

// EncryptSensitiveFields encrypts all sensitive fields in a map
func (cm *CryptoManager) EncryptSensitiveFields(data map[string]interface{}) error {
	sensitiveFields := []string{
		"client_secret",
		"admin_api_secret",
		"signing_key",
	}

	for key, value := range data {
		if contains(sensitiveFields, key) {
			if strValue, ok := value.(string); ok && strValue != "" && !IsEncrypted(strValue) {
				encrypted, err := cm.Encrypt(strValue)
				if err != nil {
					return fmt.Errorf("failed to encrypt %s: %w", key, err)
				}
				data[key] = encrypted
			}
		}
	}

	return nil
}

// DecryptSensitiveFields decrypts all sensitive fields in a map
func (cm *CryptoManager) DecryptSensitiveFields(data map[string]interface{}) error {
	sensitiveFields := []string{
		"client_secret",
		"admin_api_secret",
		"signing_key",
	}

	for key, value := range data {
		if contains(sensitiveFields, key) {
			if strValue, ok := value.(string); ok && IsEncrypted(strValue) {
				decrypted, err := cm.Decrypt(strValue)
				if err != nil {
					return fmt.Errorf("failed to decrypt %s: %w", key, err)
				}
				data[key] = decrypted
			}
		}
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
