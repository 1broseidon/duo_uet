package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"time"
	"user_experience_toolkit/internal/config"
	samlutil "user_experience_toolkit/internal/saml"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"
	dsigtypes "github.com/russellhaering/goxmldsig/types"
)

type SAMLHandler struct {
	App     *config.Application
	SP      *saml2.SAMLServiceProvider
	Session *session.Store
	BaseURL string
}

var samlIntegrationKeyPattern = regexp.MustCompile(`/saml2/sp/([A-Z0-9]+)/`)

// extractSAMLIntegrationKey extracts the integration key from Duo SSO URLs (metadata or SSO)
// Pattern: https://sso-{account}.sso.duosecurity.com/saml2/sp/{ikey}/(metadata|sso)
// This is used as a fallback for existing SAML apps that don't have ClientID populated
func extractSAMLIntegrationKey(duoURL string) string {
	if duoURL == "" {
		return ""
	}
	matches := samlIntegrationKeyPattern.FindStringSubmatch(duoURL)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// resolveIntegrationKey attempts to determine the Duo integration key for SAML apps
// It prefers the configured ClientID but can derive the key from Duo SSO metadata URLs.
func (h *SAMLHandler) resolveIntegrationKey() string {
	if h == nil || h.App == nil {
		return ""
	}

	if h.App.ClientID != "" {
		return h.App.ClientID
	}

	type source struct {
		label string
		value string
	}

	for _, candidate := range []source{
		{label: "IDP Entity ID", value: h.App.IDPEntityID},
		{label: "IDP SSO URL", value: h.App.IDPSSOURL},
		{label: "Metadata URL", value: h.App.MetadataURL},
	} {
		if key := extractSAMLIntegrationKey(candidate.value); key != "" {
			log.Printf("[SAMLHandler] Derived integration key from %s: %s", candidate.label, key)
			return key
		}
	}

	log.Printf("[SAMLHandler] Unable to determine integration key for app: %s", h.App.Name)
	return ""
}

// NewSAMLHandlerFromApp creates a SAML handler from an application configuration
func NewSAMLHandlerFromApp(app *config.Application, store *session.Store, baseURL string) (*SAMLHandler, error) {
	log.Printf("[SAMLHandler] Initializing SAML handler for app: %s (ID: %s)", app.Name, app.ID)

	// Load or generate certificates
	cert, key, err := samlutil.LoadOrGenerateCerts(app.ID, app.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificates: %w", err)
	}

	// Use stored IDP metadata from configuration
	var idpEntityID, idpSSOURL string

	if app.IDPEntityID != "" {
		idpEntityID = app.IDPEntityID
		log.Printf("[SAMLHandler] Using stored IDP Entity ID: %s", idpEntityID)
	} else {
		idpEntityID = fmt.Sprintf("https://%s", app.APIHostname)
		log.Printf("[SAMLHandler] Using default IDP Entity ID: %s", idpEntityID)
	}

	if app.IDPSSOURL != "" {
		idpSSOURL = app.IDPSSOURL
		log.Printf("[SAMLHandler] Using stored IDP SSO URL: %s", idpSSOURL)
	} else {
		idpSSOURL = fmt.Sprintf("https://%s/saml2/sso", app.APIHostname)
		log.Printf("[SAMLHandler] Using default IDP SSO URL: %s", idpSSOURL)
	}

	log.Printf("[SAMLHandler] Creating minimal IDP configuration for test utility (no signature validation)")

	// Create Service Provider
	sp, err := samlutil.NewSAMLServiceProvider(samlutil.ServiceProviderConfig{
		AppID:       app.ID,
		EntityID:    app.EntityID,
		ACSURL:      app.ACSURL,
		MetadataURL: app.MetadataURL,
		SLOURL:      fmt.Sprintf("%s/app/%s/saml/slo", baseURL, app.ID),
		Certificate: cert,
		PrivateKey:  key,
		IDPSSOURL:   idpSSOURL,
		IDPIssuer:   idpEntityID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create service provider: %w", err)
	}

	log.Printf("[SAMLHandler] SAML handler initialized successfully")

	return &SAMLHandler{
		App:     app,
		SP:      sp,
		Session: store,
		BaseURL: baseURL,
	}, nil
}

// Login displays the SAML login page
func (h *SAMLHandler) Login(c fiber.Ctx) error {
	log.Printf("[SAMLHandler] Rendering login page for app: %s", h.App.Name)

	// For backward compatibility with existing SAML apps:
	// Try ClientID first (new apps), then extract from metadata URL (existing apps)
	integrationKey := h.resolveIntegrationKey()

	return c.Render("login", fiber.Map{
		"AppType":        "saml",
		"AppID":          h.App.ID,
		"AppName":        h.App.Name,
		"EntityID":       h.App.EntityID,
		"MetadataURL":    h.App.MetadataURL,
		"APIHostname":    h.App.APIHostname,
		"AdminHostname":  getAdminHostname(h.App.APIHostname),
		"IntegrationKey": integrationKey,
	})
}

// InitiateSAML generates a SAML AuthnRequest and redirects to Duo IDP
func (h *SAMLHandler) InitiateSAML(c fiber.Ctx) error {
	log.Printf("[SAMLHandler] Initiating SAML authentication for app: %s", h.App.Name)

	// Create session to store request ID
	sess, err := h.Session.Get(c)
	if err != nil {
		log.Printf("[SAMLHandler] Failed to get session: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Session error")
	}

	// Generate AuthnRequest URL
	authURL, err := h.SP.BuildAuthURL("")
	if err != nil {
		log.Printf("[SAMLHandler] Failed to create authentication request: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to create SAML request")
	}

	// Extract request ID from the auth URL (it's in the SAMLRequest parameter)
	parsedURL, err := url.Parse(authURL)
	if err == nil {
		samlRequest := parsedURL.Query().Get("SAMLRequest")
		if samlRequest != "" {
			// Decode the SAMLRequest to extract ID
			decodedRequest, err := base64.StdEncoding.DecodeString(samlRequest)
			if err == nil {
				// Parse XML to get ID attribute
				var authnReq struct {
					ID string `xml:"ID,attr"`
				}
				if err := xml.Unmarshal(decodedRequest, &authnReq); err == nil && authnReq.ID != "" {
					// Store request ID in session
					sess.Set("saml_request_id", authnReq.ID)
					if err := sess.Save(); err != nil {
						log.Printf("[SAMLHandler] Failed to save session: %v", err)
					}

					log.Printf("[SAMLHandler] Generated AuthnRequest with ID: %s", authnReq.ID)
					log.Printf("[SAMLHandler] Session ID: %s", sess.ID())
					log.Printf("[SAMLHandler] Stored request ID in session: %s", authnReq.ID)
				}
			}
		}
	}

	log.Printf("[SAMLHandler] Redirecting to IDP: %s", authURL)
	return c.Redirect().To(authURL)
}

// ACS handles the SAML assertion consumer service (POST binding)
func (h *SAMLHandler) ACS(c fiber.Ctx) error {
	log.Printf("[SAMLHandler] Received SAML response for app: %s", h.App.Name)

	// Get session
	sess, err := h.Session.Get(c)
	if err != nil {
		log.Printf("[SAMLHandler] Failed to get session: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Session error")
	}

	log.Printf("[SAMLHandler] ACS Session ID: %s", sess.ID())

	// Get SAMLResponse from form value
	samlResponse := c.FormValue("SAMLResponse")
	if samlResponse == "" {
		log.Printf("[SAMLHandler] No SAMLResponse in form data")
		return c.Status(fiber.StatusBadRequest).SendString("Missing SAMLResponse")
	}

	log.Printf("[SAMLHandler] Received SAML Response (length: %d)", len(samlResponse))
	log.Printf("[SAMLHandler] Raw SAMLResponse (base64): %s", samlResponse)

	// Decode and log the actual SAML XML
	decodedSAML, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		log.Printf("[SAMLHandler] Failed to decode base64 SAMLResponse: %v", err)
	} else {
		log.Printf("[SAMLHandler] Decoded SAML XML:\n%s", string(decodedSAML))
	}

	log.Printf("[SAMLHandler] Parsing SAML assertion...")

	// Parse and validate the SAML response using gosaml2
	assertionInfo, err := h.SP.RetrieveAssertionInfo(samlResponse)
	if err != nil {
		log.Printf("[SAMLHandler] Failed to parse SAML response: %v", err)
		return c.Status(fiber.StatusForbidden).SendString(fmt.Sprintf("SAML validation failed: %v", err))
	}

	// Check warning info
	if assertionInfo.WarningInfo.InvalidTime {
		log.Printf("[SAMLHandler] SAML assertion has invalid time")
		return c.Status(fiber.StatusForbidden).SendString("SAML assertion time is invalid")
	}

	if assertionInfo.WarningInfo.NotInAudience {
		log.Printf("[SAMLHandler] SAML assertion audience mismatch")
		return c.Status(fiber.StatusForbidden).SendString("SAML assertion audience mismatch")
	}

	log.Printf("[SAMLHandler] SAML assertion validated successfully")

	// Extract user information
	userID := assertionInfo.NameID
	userEmail := userID // Default to NameID

	// Extract attributes from assertionInfo.Values
	attributes := make(map[string][]string)
	for key := range assertionInfo.Values {
		// Check for email attribute
		if key == "mail" || key == "email" {
			emailValue := assertionInfo.Values.Get(key)
			if emailValue != "" {
				userEmail = emailValue
			}
		}
		attributes[key] = assertionInfo.Values.GetAll(key)
	}

	// Convert attributes to JSON for session storage (avoids gob encoding issues)
	attributesJSON, err := json.Marshal(attributes)
	if err != nil {
		log.Printf("[SAMLHandler] Failed to marshal attributes: %v", err)
		attributesJSON = []byte("{}")
	}

	// Store authentication data in session
	sess.Set("authenticated", true)
	sess.Set("user_id", userID)
	sess.Set("user_email", userEmail)
	sess.Set("attributes_json", string(attributesJSON))
	sess.Set("auth_time", time.Now().Unix())

	if err := sess.Save(); err != nil {
		log.Printf("[SAMLHandler] Failed to save session: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Session error")
	}

	log.Printf("[SAMLHandler] User authenticated: %s", userEmail)

	// Redirect to success page
	return c.Redirect().To(fmt.Sprintf("/app/%s/saml/success", h.App.ID))
}

// Success renders the success page after SAML authentication
func (h *SAMLHandler) Success(c fiber.Ctx) error {
	log.Printf("[SAMLHandler] Rendering success page for app: %s", h.App.Name)

	// Get session
	sess, err := h.Session.Get(c)
	if err != nil {
		log.Printf("[SAMLHandler] Failed to get session: %v", err)
		return c.Redirect().To(fmt.Sprintf("/app/%s", h.App.ID))
	}

	// Check if authenticated
	authenticated := sess.Get("authenticated")
	if authenticated == nil || !authenticated.(bool) {
		log.Printf("[SAMLHandler] User not authenticated, redirecting to login")
		return c.Redirect().To(fmt.Sprintf("/app/%s", h.App.ID))
	}

	// Get user data
	userEmail := sess.Get("user_email")
	if userEmail == nil {
		userEmail = "Unknown"
	}

	userID := sess.Get("user_id")
	if userID == nil {
		userID = "Unknown"
	}

	authTime := sess.Get("auth_time")
	var authTimeStr string
	if authTime != nil {
		if timestamp, ok := authTime.(int64); ok {
			authTimeStr = time.Unix(timestamp, 0).Format(time.RFC3339)
		}
	}

	// Retrieve attributes from JSON stored in session
	var attributesMap map[string][]string
	attributesJSONStr := sess.Get("attributes_json")
	if attributesJSONStr != nil {
		if jsonStr, ok := attributesJSONStr.(string); ok {
			json.Unmarshal([]byte(jsonStr), &attributesMap)
		}
	}

	// Build a comprehensive response object
	responseData := map[string]interface{}{
		"nameID":     userID,
		"email":      userEmail,
		"authTime":   authTimeStr,
		"attributes": attributesMap,
	}

	// Format response data as JSON for display
	responseJSON, _ := json.MarshalIndent(responseData, "", "  ")

	// For backward compatibility with existing SAML apps:
	// Try ClientID first (new apps), then extract from metadata URL (existing apps)
	integrationKey := h.resolveIntegrationKey()

	return c.Render("success", fiber.Map{
		"AppType":        "saml",
		"AppID":          h.App.ID,
		"AppName":        h.App.Name,
		"UserEmail":      userEmail,
		"AuthFactor":     "SAML 2.0",
		"AuthResult":     "success",
		"TokenData":      string(responseJSON),
		"AttributesJSON": string(responseJSON),
		"AdminHostname":  getAdminHostname(h.App.APIHostname),
		"IntegrationKey": integrationKey,
	})
}

// Metadata serves the SP metadata XML
func (h *SAMLHandler) Metadata(c fiber.Ctx) error {
	log.Printf("[SAMLHandler] Serving metadata for app: %s", h.App.Name)

	// gosaml2 doesn't have auto-generated metadata, so we create it manually
	spDescriptor := &types.SPSSODescriptor{
		AuthnRequestsSigned:        h.SP.SignAuthnRequests,
		WantAssertionsSigned:       true,
		ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
		AssertionConsumerServices: []types.IndexedEndpoint{
			{
				Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
				Location: h.SP.AssertionConsumerServiceURL,
				Index:    1,
			},
		},
	}

	// Add certificate if available
	if h.SP.SPKeyStore != nil {
		// Try to get the certificate from the keystore
		if tlsStore, ok := h.SP.SPKeyStore.(dsig.TLSCertKeyStore); ok {
			_, certDER, err := tlsStore.GetKeyPair()
			if err == nil && len(certDER) > 0 {
				// certDER is already in DER format, encode to base64
				certData := base64.StdEncoding.EncodeToString(certDER)
				spDescriptor.KeyDescriptors = []types.KeyDescriptor{
					{
						Use: "signing",
						KeyInfo: dsigtypes.KeyInfo{
							X509Data: dsigtypes.X509Data{
								X509Certificates: []dsigtypes.X509Certificate{
									{Data: certData},
								},
							},
						},
					},
				}
			}
		}
	}

	metadata := &types.EntityDescriptor{
		EntityID:        h.SP.ServiceProviderIssuer,
		SPSSODescriptor: spDescriptor,
	}

	// Marshal to XML
	metadataXML, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		log.Printf("[SAMLHandler] Failed to marshal metadata: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to generate metadata")
	}

	// Add XML header
	fullXML := []byte(xml.Header + string(metadataXML))

	// Set content type and return XML
	c.Set("Content-Type", "application/samlmetadata+xml")
	return c.Send(fullXML)
}

// SLO handles Single Logout requests
func (h *SAMLHandler) SLO(c fiber.Ctx) error {
	log.Printf("[SAMLHandler] Handling SLO request for app: %s", h.App.Name)

	// Get session
	sess, err := h.Session.Get(c)
	if err == nil {
		// Destroy session
		if err := sess.Destroy(); err != nil {
			log.Printf("[SAMLHandler] Failed to destroy session: %v", err)
		}
	}

	// Redirect to login page
	return c.Redirect().To(fmt.Sprintf("/app/%s", h.App.ID))
}

// GetSPCertificate returns the SP's certificate in PEM format
func (h *SAMLHandler) GetSPCertificate() string {
	// Try to get certificate from keystore
	if h.SP.SPKeyStore != nil {
		if tlsStore, ok := h.SP.SPKeyStore.(dsig.TLSCertKeyStore); ok {
			_, certDER, err := tlsStore.GetKeyPair()
			if err == nil && len(certDER) > 0 {
				// Encode DER to PEM
				certPEM := pem.EncodeToMemory(&pem.Block{
					Type:  "CERTIFICATE",
					Bytes: certDER,
				})
				return string(certPEM)
			}
		}
	}
	return ""
}

// GetSPCertificateFingerprint returns the SHA256 fingerprint of the SP certificate
func (h *SAMLHandler) GetSPCertificateFingerprint() string {
	certPEM := h.GetSPCertificate()
	if certPEM == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(certPEM))
	return fmt.Sprintf("%x", hash)
}
