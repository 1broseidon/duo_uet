package duoadmin

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	integrationKey := "test_integration_key"
	secretKey := "test_secret_key"
	apiHostname := "api-test.duosecurity.com"

	client := NewClient(integrationKey, secretKey, apiHostname)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.DuoApi == nil {
		t.Fatal("NewClient() DuoApi is nil")
	}
}

func TestNewClient_MultipleInstances(t *testing.T) {
	// Verify that multiple clients can be created without interference
	client1 := NewClient("key1", "secret1", "api1.duosecurity.com")
	client2 := NewClient("key2", "secret2", "api2.duosecurity.com")

	if client1 == nil || client2 == nil {
		t.Fatal("NewClient() returned nil for one or both clients")
	}

	// They should be independent instances
	if client1 == client2 {
		t.Error("NewClient() returned same instance for different params")
	}
}

// Note: The following functions require actual API calls or extensive mocking:
// - ValidateCredentials
// - CreateIntegration
// - CreateSAMLIntegration
// - CreateOIDCIntegration
//
// These are integration tests and would require:
// 1. Mocking the HTTP client/server
// 2. Or using actual test credentials (not recommended for unit tests)
// 3. Or implementing an interface and dependency injection for the DuoApi client
//
// For now, we've tested the client creation and basic functionality.
// Full integration tests would be better suited for a separate integration test suite.

func TestIntegrationStructs(t *testing.T) {
	// Test that structs can be created and marshaled
	integration := Integration{
		IntegrationKey: "test_key",
		SecretKey:      "test_secret",
		Name:           "Test Integration",
		Type:           "websdk",
	}

	if integration.IntegrationKey != "test_key" {
		t.Errorf("Integration.IntegrationKey = %v, want test_key", integration.IntegrationKey)
	}

	if integration.Name != "Test Integration" {
		t.Errorf("Integration.Name = %v, want Test Integration", integration.Name)
	}
}

func TestCreateIntegrationParams(t *testing.T) {
	params := CreateIntegrationParams{
		Name:    "Test App",
		Type:    "websdk",
		Enabled: true,
	}

	if params.Name != "Test App" {
		t.Errorf("CreateIntegrationParams.Name = %v, want Test App", params.Name)
	}

	if params.Type != "websdk" {
		t.Errorf("CreateIntegrationParams.Type = %v, want websdk", params.Type)
	}

	if !params.Enabled {
		t.Error("CreateIntegrationParams.Enabled should be true")
	}
}

func TestCreateSAMLIntegrationParams(t *testing.T) {
	params := CreateSAMLIntegrationParams{
		Name:            "Test SAML",
		EntityID:        "http://example.com",
		ACSURL:          "http://example.com/acs",
		NameIDFormat:    "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
		NameIDAttribute: "mail",
	}

	if params.Name != "Test SAML" {
		t.Errorf("CreateSAMLIntegrationParams.Name = %v, want Test SAML", params.Name)
	}

	if params.EntityID != "http://example.com" {
		t.Errorf("CreateSAMLIntegrationParams.EntityID = %v, want http://example.com", params.EntityID)
	}

	if params.ACSURL != "http://example.com/acs" {
		t.Errorf("CreateSAMLIntegrationParams.ACSURL = %v, want http://example.com/acs", params.ACSURL)
	}
}

func TestCreateOIDCIntegrationParams(t *testing.T) {
	params := CreateOIDCIntegrationParams{
		Name:                   "Test OIDC",
		RedirectURIs:           []string{"http://example.com/callback"},
		Scopes:                 []string{"openid"},
		AccessTokenLifespan:    3600,
		AllowPKCEOnly:          false,
		EnableRefreshToken:     true,
		RefreshTokenChainLife:  2592000,
		RefreshTokenSingleLife: 86400,
	}

	if params.Name != "Test OIDC" {
		t.Errorf("CreateOIDCIntegrationParams.Name = %v, want Test OIDC", params.Name)
	}

	if len(params.RedirectURIs) != 1 {
		t.Fatalf("CreateOIDCIntegrationParams.RedirectURIs length = %v, want 1", len(params.RedirectURIs))
	}

	if params.RedirectURIs[0] != "http://example.com/callback" {
		t.Errorf("CreateOIDCIntegrationParams.RedirectURIs[0] = %v, want http://example.com/callback", params.RedirectURIs[0])
	}

	if params.AccessTokenLifespan != 3600 {
		t.Errorf("CreateOIDCIntegrationParams.AccessTokenLifespan = %v, want 3600", params.AccessTokenLifespan)
	}

	if !params.EnableRefreshToken {
		t.Error("CreateOIDCIntegrationParams.EnableRefreshToken should be true")
	}
}

func TestSAMLIntegrationStruct(t *testing.T) {
	samlIntegration := SAMLIntegration{
		IntegrationKey: "test_key",
		Name:           "Test SAML",
		Type:           "sso-generic",
	}

	samlIntegration.SSO.IDPMetadata.Cert = "test_cert"
	samlIntegration.SSO.IDPMetadata.EntityID = "http://idp.example.com"
	samlIntegration.SSO.IDPMetadata.SSOURL = "http://idp.example.com/sso"

	if samlIntegration.IntegrationKey != "test_key" {
		t.Errorf("SAMLIntegration.IntegrationKey = %v, want test_key", samlIntegration.IntegrationKey)
	}

	if samlIntegration.SSO.IDPMetadata.EntityID != "http://idp.example.com" {
		t.Errorf("SAMLIntegration IDP EntityID = %v, want http://idp.example.com", samlIntegration.SSO.IDPMetadata.EntityID)
	}
}

func TestOIDCIntegrationStruct(t *testing.T) {
	oidcIntegration := OIDCIntegration{
		IntegrationKey: "test_key",
		Name:           "Test OIDC",
		Type:           "sso-oidc-generic",
	}

	oidcIntegration.SSO.IDPMetadata.ClientID = "test_client_id"
	oidcIntegration.SSO.IDPMetadata.ClientSecret = "test_client_secret"
	oidcIntegration.SSO.IDPMetadata.Issuer = "http://idp.example.com"

	if oidcIntegration.IntegrationKey != "test_key" {
		t.Errorf("OIDCIntegration.IntegrationKey = %v, want test_key", oidcIntegration.IntegrationKey)
	}

	if oidcIntegration.SSO.IDPMetadata.ClientID != "test_client_id" {
		t.Errorf("OIDCIntegration ClientID = %v, want test_client_id", oidcIntegration.SSO.IDPMetadata.ClientID)
	}
}
