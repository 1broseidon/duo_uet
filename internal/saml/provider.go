package saml

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"

	saml2 "github.com/russellhaering/gosaml2"
	dsig "github.com/russellhaering/goxmldsig"
)

// ServiceProviderConfig holds configuration for creating a SAML Service Provider
type ServiceProviderConfig struct {
	AppID       string
	EntityID    string
	ACSURL      string
	MetadataURL string
	SLOURL      string
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
	IDPMetadata interface{} // Not used in gosaml2, kept for backward compatibility
	IDPSSOURL   string
	IDPIssuer   string
}

// NewSAMLServiceProvider creates a new SAML Service Provider instance using gosaml2
func NewSAMLServiceProvider(config ServiceProviderConfig) (*saml2.SAMLServiceProvider, error) {
	// Convert x509.Certificate and private key to tls.Certificate
	tlsCert := tls.Certificate{
		Certificate: [][]byte{config.Certificate.Raw},
		PrivateKey:  config.PrivateKey,
		Leaf:        config.Certificate,
	}

	// Create keystore from tls.Certificate (TLSCertKeyStore is a type alias for tls.Certificate)
	certStore := dsig.TLSCertKeyStore(tlsCert)

	// Create empty IDP certificate store (skip validation for test utility)
	idpCertStore := dsig.MemoryX509CertificateStore{
		Roots: []*x509.Certificate{},
	}

	// Create Service Provider
	sp := &saml2.SAMLServiceProvider{
		IdentityProviderSSOURL:      config.IDPSSOURL,
		IdentityProviderIssuer:      config.IDPIssuer,
		ServiceProviderIssuer:       config.EntityID,
		AssertionConsumerServiceURL: config.ACSURL,
		SignAuthnRequests:           true,
		AudienceURI:                 config.EntityID,
		IDPCertificateStore:         &idpCertStore,
		SPKeyStore:                  certStore,
		SkipSignatureValidation:     true, // Skip cert validation for testing tool
		AllowMissingAttributes:      true, // Allow SAML responses without AttributeStatement
	}

	return sp, nil
}
