package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"user_experience_toolkit/internal/crypto"
)

func TestEncryptionDisabledByDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialContent := `
applications:
  - id: "test-app"
    name: "Test App"
    type: "websdk"
    enabled: true
    client_id: "test_client"
    client_secret: "my-plaintext-secret"
    api_hostname: "api-test.duosecurity.com"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Encryption should be disabled by default
	if cfg.EncryptionEnabled {
		t.Error("Encryption should be disabled by default")
	}

	// Secret should remain plaintext
	if cfg.Applications[0].ClientSecret != "my-plaintext-secret" {
		t.Errorf("Client secret should be plaintext, got: %s", cfg.Applications[0].ClientSecret)
	}

	// Save should write plaintext
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Read file and verify plaintext
	data, _ := os.ReadFile(configPath)
	if !strings.Contains(string(data), "my-plaintext-secret") {
		t.Error("Config file should contain plaintext secret when encryption is disabled")
	}
}

func TestEncryptionEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Set master key for testing
	os.Setenv("UET_MASTER_KEY", "test-master-key-for-testing-12345")
	defer os.Unsetenv("UET_MASTER_KEY")

	initialContent := `
encryption_enabled: true
applications:
  - id: "test-app"
    name: "Test App"
    type: "websdk"
    enabled: true
    client_id: "test_client"
    client_secret: "my-plaintext-secret"
    api_hostname: "api-test.duosecurity.com"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config (should decrypt - but input is plaintext so passes through)
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Encryption should be enabled
	if !cfg.EncryptionEnabled {
		t.Error("Encryption should be enabled")
	}

	// In-memory secret should be plaintext (decrypted)
	if cfg.Applications[0].ClientSecret != "my-plaintext-secret" {
		t.Errorf("In-memory client secret = %s, want my-plaintext-secret", cfg.Applications[0].ClientSecret)
	}

	// Save should encrypt
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Read file and verify encrypted
	data, _ := os.ReadFile(configPath)
	fileContent := string(data)

	// Should NOT contain plaintext secret
	if strings.Contains(fileContent, "my-plaintext-secret") {
		t.Error("Config file should NOT contain plaintext secret when encryption is enabled")
	}

	// Should contain encryption marker
	if !strings.Contains(fileContent, "ENC[AES256_GCM,") {
		t.Error("Config file should contain encrypted secret marker")
	}

	// Reload and verify decryption works
	cfg2, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() after save error = %v", err)
	}

	if cfg2.Applications[0].ClientSecret != "my-plaintext-secret" {
		t.Errorf("Decrypted secret = %s, want my-plaintext-secret", cfg2.Applications[0].ClientSecret)
	}
}

func TestEncryptionRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Set master key
	os.Setenv("UET_MASTER_KEY", "test-master-key-for-testing-12345")
	defer os.Unsetenv("UET_MASTER_KEY")

	initialContent := `
encryption_enabled: true
tenants:
  - id: "test-tenant"
    name: "Test Tenant"
    admin_api_key: "admin_key"
    admin_api_secret: "admin_secret_123"
    api_hostname: "api-test.duosecurity.com"
applications:
  - id: "test-app"
    name: "Test App"
    type: "websdk"
    enabled: true
    client_id: "test_client"
    client_secret: "app_secret_456"
    api_hostname: "api-test.duosecurity.com"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify in-memory values are plaintext
	if cfg.Tenants[0].AdminAPISecret != "admin_secret_123" {
		t.Errorf("Tenant secret = %s, want admin_secret_123", cfg.Tenants[0].AdminAPISecret)
	}
	if cfg.Applications[0].ClientSecret != "app_secret_456" {
		t.Errorf("App secret = %s, want app_secret_456", cfg.Applications[0].ClientSecret)
	}

	// Save
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Reload
	cfg2, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Second LoadConfig() error = %v", err)
	}

	// Verify values survived round-trip
	if cfg2.Tenants[0].AdminAPISecret != "admin_secret_123" {
		t.Errorf("After round-trip, tenant secret = %s, want admin_secret_123", cfg2.Tenants[0].AdminAPISecret)
	}
	if cfg2.Applications[0].ClientSecret != "app_secret_456" {
		t.Errorf("After round-trip, app secret = %s, want app_secret_456", cfg2.Applications[0].ClientSecret)
	}
}

func TestEncryptionWithPreEncryptedSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Set master key
	masterKey := "test-master-key-for-testing-12345"
	os.Setenv("UET_MASTER_KEY", masterKey)
	defer os.Unsetenv("UET_MASTER_KEY")

	// Pre-encrypt a secret
	cm := crypto.NewCryptoManagerWithKey(masterKey)
	encryptedSecret, err := cm.Encrypt("my-secret-value")
	if err != nil {
		t.Fatalf("Failed to pre-encrypt: %v", err)
	}

	// Create config with already-encrypted secret
	initialContent := `
encryption_enabled: true
applications:
  - id: "test-app"
    name: "Test App"
    type: "websdk"
    enabled: true
    client_id: "test_client"
    client_secret: "` + encryptedSecret + `"
    api_hostname: "api-test.duosecurity.com"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load should decrypt
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Should be decrypted in memory
	if cfg.Applications[0].ClientSecret != "my-secret-value" {
		t.Errorf("Decrypted secret = %s, want my-secret-value", cfg.Applications[0].ClientSecret)
	}
}

func TestToggleEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Set master key
	os.Setenv("UET_MASTER_KEY", "test-master-key-for-testing-12345")
	defer os.Unsetenv("UET_MASTER_KEY")

	// Start with encryption disabled
	initialContent := `
encryption_enabled: false
applications:
  - id: "test-app"
    name: "Test App"
    type: "websdk"
    enabled: true
    client_id: "test_client"
    client_secret: "plaintext-secret"
    api_hostname: "api-test.duosecurity.com"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load and save (should stay plaintext)
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, _ := os.ReadFile(configPath)
	if !strings.Contains(string(data), "plaintext-secret") {
		t.Error("With encryption disabled, secret should remain plaintext")
	}

	// Now enable encryption
	cfg.EncryptionEnabled = true
	cm, err := crypto.NewCryptoManager()
	if err != nil {
		t.Fatalf("Failed to create crypto manager: %v", err)
	}
	cfg.cryptoManager = cm

	// Save should encrypt
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() with encryption enabled error = %v", err)
	}

	// Verify encrypted
	data, _ = os.ReadFile(configPath)
	fileContent := string(data)
	if strings.Contains(fileContent, "plaintext-secret") {
		t.Error("After enabling encryption, secret should be encrypted")
	}
	if !strings.Contains(fileContent, "ENC[AES256_GCM,") {
		t.Error("After enabling encryption, file should contain encrypted marker")
	}
}

func TestEncryptionWithEmptySecrets(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	os.Setenv("UET_MASTER_KEY", "test-master-key-for-testing-12345")
	defer os.Unsetenv("UET_MASTER_KEY")

	initialContent := `
encryption_enabled: true
applications:
  - id: "test-app"
    name: "Test App"
    type: "saml"
    enabled: true
    client_id: ""
    client_secret: ""
    api_hostname: "api-test.duosecurity.com"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Should not error on empty secrets
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() with empty secrets error = %v", err)
	}

	// Should not error on save
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() with empty secrets error = %v", err)
	}
}

func TestEncryptionWithoutMasterKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Make sure no master key in env
	os.Unsetenv("UET_MASTER_KEY")

	initialContent := `
encryption_enabled: true
applications:
  - id: "test-app"
    name: "Test App"
    type: "websdk"
    enabled: true
    client_id: "test_client"
    client_secret: "my-secret"
    api_hostname: "api-test.duosecurity.com"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Should auto-generate key file
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() without explicit key error = %v", err)
	}

	// Should still work
	if cfg.Applications[0].ClientSecret != "my-secret" {
		t.Error("Should decrypt with auto-generated key")
	}

	// Clean up auto-generated key
	os.Remove(".uet_key")
}
