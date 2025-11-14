package crypto

import (
	"strings"
	"testing"
)

func TestNewCryptoManagerWithKey(t *testing.T) {
	cm := NewCryptoManagerWithKey("test-password")
	if cm == nil {
		t.Fatal("NewCryptoManagerWithKey returned nil")
	}
	if len(cm.masterKey) != 32 {
		t.Errorf("Master key length = %d, want 32", len(cm.masterKey))
	}
}

func TestEncryptDecrypt(t *testing.T) {
	cm := NewCryptoManagerWithKey("test-password")

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple secret", "my-secret-key-12345"},
		{"duo api secret", "abcdefghijklmnopqrstuvwxyz1234567890abcd"},
		{"empty string", ""},
		{"special chars", "P@ssw0rd!#$%^&*()"},
		{"unicode", "密碼🔐"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := cm.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Empty string should return empty
			if tt.plaintext == "" {
				if encrypted != "" {
					t.Errorf("Encrypt('') = %v, want empty string", encrypted)
				}
				return
			}

			// Check format
			if !strings.HasPrefix(encrypted, "ENC[AES256_GCM,") {
				t.Errorf("Encrypted value doesn't have correct prefix: %s", encrypted)
			}

			// Decrypt
			decrypted, err := cm.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Verify
			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestDecryptPlaintext(t *testing.T) {
	cm := NewCryptoManagerWithKey("test-password")

	plaintext := "not-encrypted-value"
	decrypted, err := cm.Decrypt(plaintext)
	if err != nil {
		t.Errorf("Decrypt() of plaintext should not error: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Decrypt() should return plaintext as-is, got %v", decrypted)
	}
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		field string
		want  bool
	}{
		{"ENC[AES256_GCM,abc,def]", true},
		{"normal-value", false},
		{"", false},
		{"ENC[something-else]", false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if got := IsEncrypted(tt.field); got != tt.want {
				t.Errorf("IsEncrypted(%v) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}

func TestEncryptSensitiveFields(t *testing.T) {
	cm := NewCryptoManagerWithKey("test-password")

	data := map[string]interface{}{
		"name":             "Test App",
		"client_id":        "DI123456789",
		"client_secret":    "my-secret-123",
		"admin_api_secret": "admin-secret-456",
		"api_hostname":     "api.duosecurity.com",
	}

	err := cm.EncryptSensitiveFields(data)
	if err != nil {
		t.Fatalf("EncryptSensitiveFields() error = %v", err)
	}

	// Non-sensitive fields should be unchanged
	if data["name"] != "Test App" {
		t.Error("Non-sensitive field 'name' was changed")
	}
	if data["client_id"] != "DI123456789" {
		t.Error("Non-sensitive field 'client_id' was changed")
	}

	// Sensitive fields should be encrypted
	clientSecret, ok := data["client_secret"].(string)
	if !ok || !IsEncrypted(clientSecret) {
		t.Error("client_secret should be encrypted")
	}

	adminSecret, ok := data["admin_api_secret"].(string)
	if !ok || !IsEncrypted(adminSecret) {
		t.Error("admin_api_secret should be encrypted")
	}
}

func TestDecryptSensitiveFields(t *testing.T) {
	cm := NewCryptoManagerWithKey("test-password")

	// First encrypt
	original := map[string]interface{}{
		"client_secret":    "my-secret-123",
		"admin_api_secret": "admin-secret-456",
	}

	data := make(map[string]interface{})
	for k, v := range original {
		data[k] = v
	}

	err := cm.EncryptSensitiveFields(data)
	if err != nil {
		t.Fatalf("EncryptSensitiveFields() error = %v", err)
	}

	// Now decrypt
	err = cm.DecryptSensitiveFields(data)
	if err != nil {
		t.Fatalf("DecryptSensitiveFields() error = %v", err)
	}

	// Verify decryption
	if data["client_secret"] != original["client_secret"] {
		t.Errorf("client_secret = %v, want %v", data["client_secret"], original["client_secret"])
	}
	if data["admin_api_secret"] != original["admin_api_secret"] {
		t.Errorf("admin_api_secret = %v, want %v", data["admin_api_secret"], original["admin_api_secret"])
	}
}

func TestDifferentKeys(t *testing.T) {
	cm1 := NewCryptoManagerWithKey("password1")
	cm2 := NewCryptoManagerWithKey("password2")

	plaintext := "secret-data"

	// Encrypt with first key
	encrypted, err := cm1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Try to decrypt with second key (should fail)
	_, err = cm2.Decrypt(encrypted)
	if err == nil {
		t.Error("Decrypt() with wrong key should fail")
	}

	// Decrypt with correct key (should succeed)
	decrypted, err := cm1.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() with correct key error = %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Decrypt() = %v, want %v", decrypted, plaintext)
	}
}

func TestEncryptionDeterminism(t *testing.T) {
	cm := NewCryptoManagerWithKey("test-password")

	plaintext := "my-secret"

	// Encrypt twice
	encrypted1, _ := cm.Encrypt(plaintext)
	encrypted2, _ := cm.Encrypt(plaintext)

	// Should be different (due to random nonce)
	if encrypted1 == encrypted2 {
		t.Error("Two encryptions of same plaintext should produce different ciphertext")
	}

	// But both should decrypt to same value
	decrypted1, _ := cm.Decrypt(encrypted1)
	decrypted2, _ := cm.Decrypt(encrypted2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both encrypted values should decrypt to original plaintext")
	}
}
