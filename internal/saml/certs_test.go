package saml

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestGenerateSelfSignedCert(t *testing.T) {
	commonName := "test.example.com"

	cert, key, err := GenerateSelfSignedCert(commonName)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}

	if cert == nil {
		t.Fatal("GenerateSelfSignedCert() returned nil certificate")
	}

	if key == nil {
		t.Fatal("GenerateSelfSignedCert() returned nil private key")
	}

	// Verify certificate properties
	if cert.Subject.CommonName != commonName {
		t.Errorf("Certificate CommonName = %v, want %v", cert.Subject.CommonName, commonName)
	}

	if len(cert.Subject.Organization) == 0 {
		t.Error("Certificate should have organization")
	}

	// Verify key usage
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Error("Certificate should have DigitalSignature key usage")
	}

	// Verify it's valid for 10 years
	duration := cert.NotAfter.Sub(cert.NotBefore)
	expectedDuration := 10 * 365 * 24 * 60 * 60 // 10 years in seconds
	if int(duration.Seconds()) < expectedDuration-3600 { // Allow 1 hour variance
		t.Errorf("Certificate validity period is too short: %v", duration)
	}

	// Verify private key size
	if key.Size() != 256 { // 2048 bits = 256 bytes
		t.Errorf("Private key size = %v, want 256", key.Size())
	}
}

func TestLoadOrGenerateCerts_Generate(t *testing.T) {
	tmpDir := t.TempDir()

	// Override certs directory for testing
	originalGetwd, _ := os.Getwd()
	defer os.Chdir(originalGetwd)

	// Create a temporary working directory
	testWorkDir := tmpDir
	os.Chdir(testWorkDir)

	appID := "test-app-123"
	commonName := "test.example.com"

	cert, key, err := LoadOrGenerateCerts(appID, commonName)
	if err != nil {
		t.Fatalf("LoadOrGenerateCerts() error = %v", err)
	}

	if cert == nil || key == nil {
		t.Fatal("LoadOrGenerateCerts() returned nil certificate or key")
	}

	// Verify files were created
	certsDir := "./certs"
	certPath := filepath.Join(certsDir, "saml-"+appID+".cert")
	keyPath := filepath.Join(certsDir, "saml-"+appID+".key")

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Errorf("Certificate file was not created at %s", certPath)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Errorf("Private key file was not created at %s", keyPath)
	}
}

func TestLoadOrGenerateCerts_Load(t *testing.T) {
	tmpDir := t.TempDir()
	originalGetwd, _ := os.Getwd()
	defer os.Chdir(originalGetwd)

	testWorkDir := tmpDir
	os.Chdir(testWorkDir)

	appID := "test-app-456"
	commonName := "test.example.com"

	// First call - generate
	cert1, key1, err := LoadOrGenerateCerts(appID, commonName)
	if err != nil {
		t.Fatalf("First LoadOrGenerateCerts() error = %v", err)
	}

	// Second call - should load existing
	cert2, key2, err := LoadOrGenerateCerts(appID, commonName)
	if err != nil {
		t.Fatalf("Second LoadOrGenerateCerts() error = %v", err)
	}

	// Certificates should be identical
	if !cert1.Equal(cert2) {
		t.Error("Loaded certificate differs from original")
	}

	// Compare key modulus to verify it's the same key
	if key1.N.Cmp(key2.N) != 0 {
		t.Error("Loaded private key differs from original")
	}
}

func TestCertToPEM(t *testing.T) {
	cert, _, err := GenerateSelfSignedCert("test.example.com")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}

	pem := CertToPEM(cert)

	if pem == "" {
		t.Error("CertToPEM() returned empty string")
	}

	// Verify PEM format
	if pem[:27] != "-----BEGIN CERTIFICATE-----" {
		t.Error("CertToPEM() did not return valid PEM format")
	}

	// PEM has a newline at the end, so we need to trim or check differently
	if !contains(pem, "-----END CERTIFICATE-----") {
		t.Error("CertToPEM() did not contain correct PEM end marker")
	}
}

func TestKeyToPEM(t *testing.T) {
	_, key, err := GenerateSelfSignedCert("test.example.com")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}

	pem := KeyToPEM(key)

	if pem == "" {
		t.Error("KeyToPEM() returned empty string")
	}

	// Verify PEM format
	if pem[:31] != "-----BEGIN RSA PRIVATE KEY-----" {
		t.Error("KeyToPEM() did not return valid PEM format")
	}

	// PEM has a newline at the end, so we need to trim or check differently
	if !contains(pem, "-----END RSA PRIVATE KEY-----") {
		t.Error("KeyToPEM() did not contain correct PEM end marker")
	}
}

func TestSaveCertsToDisk(t *testing.T) {
	tmpDir := t.TempDir()

	cert, key, err := GenerateSelfSignedCert("test.example.com")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}

	certPath := filepath.Join(tmpDir, "test.cert")
	keyPath := filepath.Join(tmpDir, "test.key")

	err = saveCertsToDisk(cert, key, certPath, keyPath)
	if err != nil {
		t.Errorf("saveCertsToDisk() error = %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Error("Certificate file was not created")
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("Private key file was not created")
	}

	// Verify file permissions
	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Failed to stat key file: %v", err)
	}

	if keyInfo.Mode().Perm() != 0600 {
		t.Errorf("Private key file has wrong permissions: %o, want 0600", keyInfo.Mode().Perm())
	}
}

func TestLoadCertsFromDisk(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate and save certificates
	originalCert, originalKey, err := GenerateSelfSignedCert("test.example.com")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}

	certPath := filepath.Join(tmpDir, "test.cert")
	keyPath := filepath.Join(tmpDir, "test.key")

	err = saveCertsToDisk(originalCert, originalKey, certPath, keyPath)
	if err != nil {
		t.Fatalf("saveCertsToDisk() error = %v", err)
	}

	// Load certificates
	loadedCert, loadedKey, err := loadCertsFromDisk(certPath, keyPath)
	if err != nil {
		t.Fatalf("loadCertsFromDisk() error = %v", err)
	}

	// Verify certificates match
	if !originalCert.Equal(loadedCert) {
		t.Error("Loaded certificate differs from original")
	}

	// Verify keys match
	if originalKey.N.Cmp(loadedKey.N) != 0 {
		t.Error("Loaded private key differs from original")
	}
}

func TestLoadCertsFromDisk_Errors(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		setup   func() (string, string)
		wantErr bool
	}{
		{
			name: "nonexistent cert file",
			setup: func() (string, string) {
				return filepath.Join(tmpDir, "nonexistent.cert"), filepath.Join(tmpDir, "nonexistent.key")
			},
			wantErr: true,
		},
		{
			name: "invalid cert PEM",
			setup: func() (string, string) {
				certPath := filepath.Join(tmpDir, "invalid.cert")
				keyPath := filepath.Join(tmpDir, "invalid.key")
				os.WriteFile(certPath, []byte("not a valid PEM"), 0644)
				os.WriteFile(keyPath, []byte("not a valid PEM"), 0644)
				return certPath, keyPath
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certPath, keyPath := tt.setup()
			_, _, err := loadCertsFromDisk(certPath, keyPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadCertsFromDisk() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
