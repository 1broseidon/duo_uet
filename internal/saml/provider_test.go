package saml

import (
	"testing"
)

func TestNewSAMLServiceProvider(t *testing.T) {
	// Generate test certificates
	cert, key, err := GenerateSelfSignedCert("test.example.com")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}

	config := ServiceProviderConfig{
		AppID:       "test-app",
		EntityID:    "http://example.com/entity",
		ACSURL:      "http://example.com/acs",
		MetadataURL: "http://example.com/metadata",
		SLOURL:      "http://example.com/slo",
		Certificate: cert,
		PrivateKey:  key,
		IDPSSOURL:   "http://idp.example.com/sso",
		IDPIssuer:   "http://idp.example.com",
	}

	sp, err := NewSAMLServiceProvider(config)
	if err != nil {
		t.Fatalf("NewSAMLServiceProvider() error = %v", err)
	}

	if sp == nil {
		t.Fatal("NewSAMLServiceProvider() returned nil")
	}

	// Verify configuration
	if sp.ServiceProviderIssuer != config.EntityID {
		t.Errorf("ServiceProviderIssuer = %v, want %v", sp.ServiceProviderIssuer, config.EntityID)
	}

	if sp.AssertionConsumerServiceURL != config.ACSURL {
		t.Errorf("AssertionConsumerServiceURL = %v, want %v", sp.AssertionConsumerServiceURL, config.ACSURL)
	}

	if sp.IdentityProviderSSOURL != config.IDPSSOURL {
		t.Errorf("IdentityProviderSSOURL = %v, want %v", sp.IdentityProviderSSOURL, config.IDPSSOURL)
	}

	if sp.IdentityProviderIssuer != config.IDPIssuer {
		t.Errorf("IdentityProviderIssuer = %v, want %v", sp.IdentityProviderIssuer, config.IDPIssuer)
	}

	if !sp.SignAuthnRequests {
		t.Error("SignAuthnRequests should be true")
	}

	if !sp.SkipSignatureValidation {
		t.Error("SkipSignatureValidation should be true for testing")
	}

	if !sp.AllowMissingAttributes {
		t.Error("AllowMissingAttributes should be true")
	}

	if sp.AudienceURI != config.EntityID {
		t.Errorf("AudienceURI = %v, want %v", sp.AudienceURI, config.EntityID)
	}

	// Verify key store is set
	if sp.SPKeyStore == nil {
		t.Error("SPKeyStore should not be nil")
	}

	// Verify IDP certificate store is set
	if sp.IDPCertificateStore == nil {
		t.Error("IDPCertificateStore should not be nil")
	}
}

func TestNewSAMLServiceProvider_MultipleInstances(t *testing.T) {
	// Test that multiple service providers can be created with different configs
	cert1, key1, err := GenerateSelfSignedCert("test1.example.com")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}

	cert2, key2, err := GenerateSelfSignedCert("test2.example.com")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}

	config1 := ServiceProviderConfig{
		AppID:       "app1",
		EntityID:    "http://app1.example.com/entity",
		ACSURL:      "http://app1.example.com/acs",
		MetadataURL: "http://app1.example.com/metadata",
		SLOURL:      "http://app1.example.com/slo",
		Certificate: cert1,
		PrivateKey:  key1,
		IDPSSOURL:   "http://idp.example.com/sso",
		IDPIssuer:   "http://idp.example.com",
	}

	config2 := ServiceProviderConfig{
		AppID:       "app2",
		EntityID:    "http://app2.example.com/entity",
		ACSURL:      "http://app2.example.com/acs",
		MetadataURL: "http://app2.example.com/metadata",
		SLOURL:      "http://app2.example.com/slo",
		Certificate: cert2,
		PrivateKey:  key2,
		IDPSSOURL:   "http://idp.example.com/sso",
		IDPIssuer:   "http://idp.example.com",
	}

	sp1, err := NewSAMLServiceProvider(config1)
	if err != nil {
		t.Fatalf("NewSAMLServiceProvider(config1) error = %v", err)
	}

	sp2, err := NewSAMLServiceProvider(config2)
	if err != nil {
		t.Fatalf("NewSAMLServiceProvider(config2) error = %v", err)
	}

	// Verify they have different configurations
	if sp1.ServiceProviderIssuer == sp2.ServiceProviderIssuer {
		t.Error("Service providers should have different entity IDs")
	}

	if sp1.AssertionConsumerServiceURL == sp2.AssertionConsumerServiceURL {
		t.Error("Service providers should have different ACS URLs")
	}
}
