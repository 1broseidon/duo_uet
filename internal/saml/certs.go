package saml

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// getCertsDir returns the certs directory path, handling Docker vs local environments
func getCertsDir() string {
	// Check if running in Docker (check for /app directory)
	if _, err := os.Stat("/app"); err == nil {
		return "/app/certs"
	}
	// Running locally
	return "./certs"
}

// GenerateSelfSignedCert generates a self-signed X.509 certificate for SAML signing
func GenerateSelfSignedCert(commonName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour) // 10 years

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Duo User Experience Toolkit"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, privateKey, nil
}

// LoadOrGenerateCerts loads certificates from disk or generates new ones
func LoadOrGenerateCerts(appID string, commonName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Use absolute path for Docker compatibility
	certsDir := getCertsDir()
	certPath := filepath.Join(certsDir, fmt.Sprintf("saml-%s.cert", appID))
	keyPath := filepath.Join(certsDir, fmt.Sprintf("saml-%s.key", appID))

	// Try to load existing certificates
	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			cert, key, err := loadCertsFromDisk(certPath, keyPath)
			if err == nil {
				return cert, key, nil
			}
			// If loading fails, generate new ones
		}
	}

	// Generate new certificates
	cert, key, err := GenerateSelfSignedCert(commonName)
	if err != nil {
		return nil, nil, err
	}

	// Create certs directory if it doesn't exist
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create certs directory: %w", err)
	}

	// Save to disk
	if err := saveCertsToDisk(cert, key, certPath, keyPath); err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

// loadCertsFromDisk loads certificate and private key from PEM files
func loadCertsFromDisk(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Load certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Load private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return cert, key, nil
}

// saveCertsToDisk saves certificate and private key to PEM files
func saveCertsToDisk(cert *x509.Certificate, key *rsa.PrivateKey, certPath, keyPath string) error {
	// Save certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Save private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

// CertToPEM converts a certificate to PEM format string
func CertToPEM(cert *x509.Certificate) string {
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}))
}

// KeyToPEM converts a private key to PEM format string
func KeyToPEM(key *rsa.PrivateKey) string {
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))
}
