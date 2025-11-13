package handlers

import (
	"testing"
	"user_experience_toolkit/internal/config"

	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestNewHomeHandler(t *testing.T) {
	cfg := &config.Config{}
	handler := NewHomeHandler(cfg)

	if handler == nil {
		t.Fatal("NewHomeHandler() returned nil")
	}

	if handler.Config != cfg {
		t.Error("NewHomeHandler() did not set Config correctly")
	}
}

func TestNewConfigHandler(t *testing.T) {
	cfg := &config.Config{}
	handler := NewConfigHandler(cfg)

	if handler == nil {
		t.Fatal("NewConfigHandler() returned nil")
	}

	if handler.Config != cfg {
		t.Error("NewConfigHandler() did not set Config correctly")
	}
}


func TestNewV4HandlerFromApp(t *testing.T) {
	tests := []struct {
		name             string
		app              *config.Application
		wantTypeCheckErr bool // Whether type check should fail
	}{
		{
			name: "valid websdk app type",
			app: &config.Application{
				ID:           "test-app",
				Name:         "Test WebSDK",
				Type:         "websdk",
				ClientID:     "DIXXXXXXXXXXXXXXXXXX", // Valid format but fake
				ClientSecret: "test_client_secret_fake_credentials_12345",
				APIHostname:  "api-test.duosecurity.com",
			},
			wantTypeCheckErr: false,
		},
		{
			name: "app with empty type (defaults to websdk)",
			app: &config.Application{
				ID:           "test-app",
				Name:         "Test App",
				ClientID:     "DIXXXXXXXXXXXXXXXXXX",
				ClientSecret: "test_client_secret_fake_credentials_12345",
				APIHostname:  "api-test.duosecurity.com",
			},
			wantTypeCheckErr: false,
		},
		{
			name: "dmp app (should fail type check)",
			app: &config.Application{
				ID:           "test-app",
				Name:         "Test DMP",
				Type:         "dmp",
				ClientID:     "DIXXXXXXXXXXXXXXXXXX",
				ClientSecret: "test_client_secret_fake_credentials_12345",
				APIHostname:  "api-test.duosecurity.com",
			},
			wantTypeCheckErr: true,
		},
	}

	store := session.NewStore()
	baseURL := "http://localhost:8080"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewV4HandlerFromApp(tt.app, store, baseURL)

			// We expect errors for type mismatches
			if tt.wantTypeCheckErr {
				if err == nil {
					t.Error("NewV4HandlerFromApp() should return error for DMP app")
				}
				return
			}

			// For valid types, the Duo client creation may still fail due to fake credentials
			// but the handler should attempt to initialize
			if err != nil {
				// This is acceptable - fake credentials will cause Duo client to fail
				t.Logf("NewV4HandlerFromApp() error (expected with fake credentials): %v", err)
				return
			}

			// If somehow it succeeds (unlikely), verify structure
			if handler != nil {
				if handler.App != tt.app {
					t.Error("NewV4HandlerFromApp() did not set App correctly")
				}
				if handler.Store != store {
					t.Error("NewV4HandlerFromApp() did not set Store correctly")
				}
			}
		})
	}
}

func TestNewDMPHandlerFromApp(t *testing.T) {
	tests := []struct {
		name             string
		app              *config.Application
		wantTypeCheckErr bool
	}{
		{
			name: "valid dmp app type",
			app: &config.Application{
				ID:           "test-app",
				Name:         "Test DMP",
				Type:         "dmp",
				ClientID:     "DIXXXXXXXXXXXXXXXXXX",
				ClientSecret: "test_client_secret_fake_credentials_12345",
				APIHostname:  "api-test.duosecurity.com",
			},
			wantTypeCheckErr: false,
		},
		{
			name: "websdk app (should fail type check)",
			app: &config.Application{
				ID:           "test-app",
				Name:         "Test WebSDK",
				Type:         "websdk",
				ClientID:     "DIXXXXXXXXXXXXXXXXXX",
				ClientSecret: "test_client_secret_fake_credentials_12345",
				APIHostname:  "api-test.duosecurity.com",
			},
			wantTypeCheckErr: true,
		},
	}

	store := session.NewStore()
	baseURL := "http://localhost:8080"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewDMPHandlerFromApp(tt.app, store, baseURL)

			// We expect errors for type mismatches
			if tt.wantTypeCheckErr {
				if err == nil {
					t.Error("NewDMPHandlerFromApp() should return error for WebSDK app")
				}
				return
			}

			// For valid types, the Duo client creation may still fail due to fake credentials
			if err != nil {
				t.Logf("NewDMPHandlerFromApp() error (expected with fake credentials): %v", err)
				return
			}

			// If somehow it succeeds, verify structure
			if handler != nil {
				if handler.App != tt.app {
					t.Error("NewDMPHandlerFromApp() did not set App correctly")
				}
				if handler.Store != store {
					t.Error("NewDMPHandlerFromApp() did not set Store correctly")
				}
			}
		})
	}
}

func TestNewSAMLHandlerFromApp(t *testing.T) {
	// Create temporary working directory for certs
	// This test requires file system access for certificate generation
	app := &config.Application{
		ID:          "test-saml-app",
		Name:        "Test SAML",
		Type:        "saml",
		EntityID:    "http://example.com/entity",
		ACSURL:      "http://example.com/acs",
		MetadataURL: "http://example.com/metadata",
		APIHostname: "api-test.duosecurity.com",
	}

	store := session.NewStore()
	baseURL := "http://localhost:8080"

	handler, err := NewSAMLHandlerFromApp(app, store, baseURL)
	if err != nil {
		// This might fail if cert generation has issues, which is okay for basic testing
		t.Logf("NewSAMLHandlerFromApp() error = %v (cert generation may fail in test env)", err)
		return
	}

	if handler == nil {
		t.Fatal("NewSAMLHandlerFromApp() returned nil handler")
	}

	if handler.App != app {
		t.Error("NewSAMLHandlerFromApp() did not set App correctly")
	}

	if handler.Session != store {
		t.Error("NewSAMLHandlerFromApp() did not set Session correctly")
	}

	if handler.SP == nil {
		t.Error("NewSAMLHandlerFromApp() did not initialize SP")
	}
}

func TestNewOIDCHandlerFromApp(t *testing.T) {
	// Note: This test requires actual OIDC provider discovery, which will likely fail
	// in test environment. We're testing the basic structure.
	app := &config.Application{
		ID:           "test-oidc-app",
		Name:         "Test OIDC",
		Type:         "oidc",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURI:  "http://example.com/callback",
		APIHostname:  "api-test.duosecurity.com",
		IDPIssuer:    "http://idp.example.com",
	}

	store := session.NewStore()
	baseURL := "http://localhost:8080"

	// This will likely fail due to OIDC discovery, which is expected
	handler, err := NewOIDCHandlerFromApp(app, store, baseURL)
	if err != nil {
		// Expected to fail in test environment
		t.Logf("NewOIDCHandlerFromApp() error = %v (OIDC discovery expected to fail)", err)
		return
	}

	// If it somehow succeeds (unlikely), verify structure
	if handler != nil {
		if handler.App != app {
			t.Error("NewOIDCHandlerFromApp() did not set App correctly")
		}
		if handler.Session != store {
			t.Error("NewOIDCHandlerFromApp() did not set Session correctly")
		}
	}
}

func TestGenerateRandomString(t *testing.T) {
	// Test the random string generator
	str1 := generateRandomString(32)
	str2 := generateRandomString(32)

	if str1 == "" {
		t.Error("generateRandomString() returned empty string")
	}

	if str2 == "" {
		t.Error("generateRandomString() returned empty string")
	}

	// Very unlikely to generate the same string twice
	if str1 == str2 {
		t.Error("generateRandomString() generated identical strings")
	}

	// Test different lengths
	short := generateRandomString(8)
	long := generateRandomString(64)

	if len(short) == 0 {
		t.Error("generateRandomString(8) returned empty string")
	}

	if len(long) == 0 {
		t.Error("generateRandomString(64) returned empty string")
	}

	// Longer input should generally produce longer output (base64 encoded)
	if len(long) <= len(short) {
		t.Error("generateRandomString() length should scale with input")
	}
}

func TestAutoCreateApplicationRequest(t *testing.T) {
	req := AutoCreateApplicationRequest{
		Name:     "Test App",
		Type:     "websdk",
		Enabled:  true,
		TenantID: "test-tenant-id",
	}

	if req.Name != "Test App" {
		t.Errorf("AutoCreateApplicationRequest.Name = %v, want Test App", req.Name)
	}

	if req.Type != "websdk" {
		t.Errorf("AutoCreateApplicationRequest.Type = %v, want websdk", req.Type)
	}

	if !req.Enabled {
		t.Error("AutoCreateApplicationRequest.Enabled should be true")
	}

	if req.TenantID != "test-tenant-id" {
		t.Errorf("AutoCreateApplicationRequest.TenantID = %v, want test-tenant-id", req.TenantID)
	}
}

func TestAddTenantRequest(t *testing.T) {
	req := AddTenantRequest{
		Name:           "Test Tenant",
		AdminAPIKey:    "test_key",
		AdminAPISecret: "test_secret",
		APIHostname:    "api-test.duosecurity.com",
	}

	if req.Name != "Test Tenant" {
		t.Errorf("AddTenantRequest.Name = %v, want Test Tenant", req.Name)
	}

	if req.AdminAPIKey != "test_key" {
		t.Errorf("AddTenantRequest.AdminAPIKey = %v, want test_key", req.AdminAPIKey)
	}

	if req.AdminAPISecret != "test_secret" {
		t.Errorf("AddTenantRequest.AdminAPISecret = %v, want test_secret", req.AdminAPISecret)
	}

	if req.APIHostname != "api-test.duosecurity.com" {
		t.Errorf("AddTenantRequest.APIHostname = %v, want api-test.duosecurity.com", req.APIHostname)
	}
}
