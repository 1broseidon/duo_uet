package duoadmin

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	duoapi "github.com/duosecurity/duo_api_golang"
)

// Client provides access to Duo's Admin API with extended functionality
type Client struct {
	*duoapi.DuoApi
}

// NewClient creates a new Duo Admin API client
func NewClient(integrationKey, secretKey, apiHostname string) *Client {
	duoClient := duoapi.NewDuoApi(
		integrationKey,
		secretKey,
		apiHostname,
		"user_experience_toolkit",
	)
	return &Client{DuoApi: duoClient}
}

// Integration represents a Duo integration/application
type Integration struct {
	IntegrationKey string `json:"integration_key"`
	SecretKey      string `json:"secret_key"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	APIHostname    string `json:"-"` // Not returned by API, we set this manually
}

// CreateIntegrationResult models the response from creating an integration
type CreateIntegrationResult struct {
	Stat     string      `json:"stat"`
	Response Integration `json:"response"`
}

// CreateIntegrationParams holds parameters for creating a new integration
type CreateIntegrationParams struct {
	Name    string
	Type    string // "websdk" for WebSDK/Universal Prompt, "device-management-portal" for DMP
	Enabled bool
}

// CreateSAMLIntegrationParams holds parameters for creating a SAML integration
type CreateSAMLIntegrationParams struct {
	Name            string
	EntityID        string
	ACSURL          string
	NameIDFormat    string
	NameIDAttribute string
}

// CreateOIDCIntegrationParams holds parameters for creating an OIDC integration
type CreateOIDCIntegrationParams struct {
	Name                   string
	RedirectURIs           []string
	Scopes                 []string // e.g., ["openid", "email", "profile"]
	AccessTokenLifespan    int      // In seconds, defaults to 3600
	AllowPKCEOnly          bool
	EnableRefreshToken     bool
	RefreshTokenChainLife  int // In seconds, defaults to 2592000
	RefreshTokenSingleLife int // In seconds, defaults to 86400
}

// SAMLIntegration represents a Duo SAML integration
type SAMLIntegration struct {
	IntegrationKey string `json:"integration_key"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	SSO            struct {
		IDPMetadata struct {
			Cert        string `json:"cert"`
			EntityID    string `json:"entity_id"`
			MetadataURL string `json:"metadata_url"`
			SLOURL      string `json:"slo_url"`
			SSOURL      string `json:"sso_url"`
		} `json:"idp_metadata"`
		SAMLConfig struct {
			ACSURLs          []interface{} `json:"acs_urls"`
			EntityID         string        `json:"entity_id"`
			NameIDAttribute  string        `json:"nameid_attribute"`
			NameIDFormat     string        `json:"nameid_format"`
			SignAssertion    bool          `json:"sign_assertion"`
			SignResponse     bool          `json:"sign_response"`
			SigningAlgorithm string        `json:"signing_algorithm"`
		} `json:"saml_config"`
	} `json:"sso"`
}

// OIDCIntegration represents a Duo OIDC integration
type OIDCIntegration struct {
	IntegrationKey string `json:"integration_key"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	SSO            struct {
		IDPMetadata struct {
			AuthorizeEndpointURL          string `json:"authorize_endpoint_url"`
			ClientID                      string `json:"client_id"`
			ClientSecret                  string `json:"client_secret"`
			DiscoveryURL                  string `json:"discovery_url"`
			Issuer                        string `json:"issuer"`
			JWKSEndpointURL               string `json:"jwks_endpoint_url"`
			TokenEndpointURL              string `json:"token_endpoint_url"`
			TokenIntrospectionEndpointURL string `json:"token_introspection_endpoint_url"`
			UserInfoEndpointURL           string `json:"userinfo_endpoint_url"`
		} `json:"idp_metadata"`
		OIDCConfig struct {
			GrantTypes struct {
				AuthorizationCode struct {
					AccessTokenLifespan int  `json:"access_token_lifespan"`
					AllowPKCEOnly       bool `json:"allow_pkce_only"`
					RefreshToken        struct {
						RefreshTokenChainLifespan  int `json:"refresh_token_chain_lifespan"`
						RefreshTokenSingleLifespan int `json:"refresh_token_single_lifespan"`
					} `json:"refresh_token,omitempty"`
				} `json:"authorization_code"`
			} `json:"grant_types"`
			RedirectURIs []string `json:"redirect_uris"`
			Scopes       []struct {
				Name   string   `json:"name"`
				Claims []string `json:"claims"`
			} `json:"scopes"`
		} `json:"oidc_config"`
	} `json:"sso"`
}

// CreateIntegration creates a new Duo integration (application) via the Admin API
// This implements POST /admin/v1/integrations
// See: https://duo.com/docs/adminapi-v1#create-integration
func (c *Client) CreateIntegration(params CreateIntegrationParams) (*Integration, error) {
	log.Printf("[DuoAdmin] Creating integration with name: %s, type: %s", params.Name, params.Type)

	// Build request parameters
	requestParams := url.Values{}
	requestParams.Set("name", params.Name)
	requestParams.Set("type", params.Type)

	// Note: Per user requirement, all users should be allowed to authenticate
	// New applications are disabled by default in Duo Admin Panel, but we want them enabled
	// However, the v1 API doesn't have a direct "enabled" parameter
	// The application will be created in active state by default

	log.Printf("[DuoAdmin] Request parameters: %v", requestParams)

	// Make the API call
	resp, body, err := c.SignedCall(
		http.MethodPost,
		"/admin/v1/integrations",
		requestParams,
		duoapi.UseTimeout,
	)
	if err != nil {
		log.Printf("[DuoAdmin] Failed to create integration: %v", err)
		return nil, fmt.Errorf("failed to create integration: %w", err)
	}

	log.Printf("[DuoAdmin] Create integration response status: %d", resp.StatusCode)
	log.Printf("[DuoAdmin] Create integration response body: %s", string(body))

	// Parse the response
	var result CreateIntegrationResult
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[DuoAdmin] Failed to parse create integration response: %v", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if the API call was successful
	if result.Stat != "OK" {
		log.Printf("[DuoAdmin] Create integration failed. Stat: %s", result.Stat)
		return nil, fmt.Errorf("API returned error status: %s", result.Stat)
	}

	// The API hostname is not included in the response, but it's the same as the client's
	// We need to return it so the caller can save the complete application config
	integration := result.Response

	log.Printf("[DuoAdmin] Integration created successfully. Integration key: %s", integration.IntegrationKey)
	return &integration, nil
}

// ValidateCredentials checks if the provided Admin API credentials are valid
// by making a simple API call
func (c *Client) ValidateCredentials() error {
	log.Printf("[DuoAdmin] Validating Admin API credentials...")

	// Make a simple API call to check if credentials are valid
	// We'll use the /admin/v1/info/summary endpoint which requires minimal permissions
	resp, body, err := c.SignedCall(
		http.MethodGet,
		"/admin/v1/info/summary",
		url.Values{},
		duoapi.UseTimeout,
	)
	if err != nil {
		log.Printf("[DuoAdmin] Credential validation failed with error: %v", err)
		return fmt.Errorf("failed to validate credentials: %w", err)
	}

	log.Printf("[DuoAdmin] Validation response status: %d", resp.StatusCode)
	log.Printf("[DuoAdmin] Validation response body: %s", string(body))

	// Check if we got a valid response
	var result struct {
		Stat    string `json:"stat"`
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[DuoAdmin] Failed to parse validation response: %v", err)
		return fmt.Errorf("failed to parse validation response: %w", err)
	}

	if result.Stat != "OK" {
		log.Printf("[DuoAdmin] Credentials validation failed. Stat: %s, Code: %d, Message: %s", result.Stat, result.Code, result.Message)
		return fmt.Errorf("credentials validation failed: %s (code: %d, message: %s)", result.Stat, result.Code, result.Message)
	}

	log.Printf("[DuoAdmin] Credentials validated successfully")
	return nil
}

// CreateSAMLIntegration creates a new Duo SAML integration (application) via the Admin API
// This implements POST /admin/v3/integrations
func (c *Client) CreateSAMLIntegration(params CreateSAMLIntegrationParams) (*SAMLIntegration, error) {
	log.Printf("[DuoAdmin] Creating SAML integration with name: %s, entity_id: %s", params.Name, params.EntityID)

	// Build the SAML configuration with only required parameters
	samlConfig := map[string]interface{}{
		"entity_id": params.EntityID,
		"acs_urls": []map[string]interface{}{
			{
				"url": params.ACSURL,
			},
		},
		"nameid_format":     params.NameIDFormat,
		"nameid_attribute":  params.NameIDAttribute,
		"sign_assertion":    true,
		"sign_response":     true,
		"signing_algorithm": "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256",
	}

	// Build the SSO configuration object with saml_config nested
	ssoConfig := map[string]interface{}{
		"saml_config": samlConfig,
	}

	// Build the main request body for v3 API using JSONParams
	jsonParams := duoapi.JSONParams{
		"name":        params.Name,
		"type":        "sso-generic",
		"user_access": "ALL_USERS",
		"sso":         ssoConfig,
	}

	log.Printf("[DuoAdmin] Request params: %+v", jsonParams)

	// Make the API call using v3 endpoint with JSONSignedCall (uses v5 signing)
	resp, body, err := c.JSONSignedCall(
		http.MethodPost,
		"/admin/v3/integrations",
		jsonParams,
		duoapi.UseTimeout,
	)
	if err != nil {
		log.Printf("[DuoAdmin] Failed to create SAML integration: %v", err)
		return nil, fmt.Errorf("failed to create SAML integration: %w", err)
	}

	log.Printf("[DuoAdmin] Create SAML integration response status: %d", resp.StatusCode)
	log.Printf("[DuoAdmin] Create SAML integration response body: %s", string(body))

	// Parse the response
	var result struct {
		Stat     string          `json:"stat"`
		Response SAMLIntegration `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[DuoAdmin] Failed to parse create SAML integration response: %v", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if the API call was successful
	if result.Stat != "OK" {
		log.Printf("[DuoAdmin] Create SAML integration failed. Stat: %s", result.Stat)
		return nil, fmt.Errorf("API returned error status: %s", result.Stat)
	}

	integration := result.Response
	log.Printf("[DuoAdmin] SAML integration created successfully. Integration key: %s", integration.IntegrationKey)
	return &integration, nil
}

// CreateOIDCIntegration creates a new Duo OIDC integration (application) via the Admin API
// This implements POST /admin/v3/integrations with type "sso-oidc-generic"
func (c *Client) CreateOIDCIntegration(params CreateOIDCIntegrationParams) (*OIDCIntegration, error) {
	log.Printf("[DuoAdmin] Creating OIDC integration with name: %s", params.Name)

	// Set defaults
	if params.AccessTokenLifespan == 0 {
		params.AccessTokenLifespan = 3600 // 1 hour
	}
	if params.EnableRefreshToken {
		if params.RefreshTokenChainLife == 0 {
			params.RefreshTokenChainLife = 2592000 // 30 days
		}
		if params.RefreshTokenSingleLife == 0 {
			params.RefreshTokenSingleLife = 86400 // 1 day
		}
	}

	// Build the authorization_code grant type configuration
	authCodeConfig := map[string]interface{}{
		"access_token_lifespan": params.AccessTokenLifespan,
		"allow_pkce_only":       params.AllowPKCEOnly,
	}

	// Add refresh token config if enabled
	if params.EnableRefreshToken {
		authCodeConfig["refresh_token"] = map[string]interface{}{
			"refresh_token_chain_lifespan":  params.RefreshTokenChainLife,
			"refresh_token_single_lifespan": params.RefreshTokenSingleLife,
		}
	}

	// Build the scopes configuration - only openid is required
	scopesList := []map[string]interface{}{
		{"name": "openid"},
	}

	// Build the OIDC configuration
	oidcConfig := map[string]interface{}{
		"grant_types": map[string]interface{}{
			"authorization_code": authCodeConfig,
		},
		"redirect_uris": params.RedirectURIs,
		"scopes":        scopesList,
	}

	// Build the SSO configuration object with oidc_config nested
	ssoConfig := map[string]interface{}{
		"oidc_config": oidcConfig,
	}

	// Build the main request body for v3 API using JSONParams
	jsonParams := duoapi.JSONParams{
		"name":        params.Name,
		"type":        "sso-oidc-generic",
		"user_access": "ALL_USERS",
		"sso":         ssoConfig,
	}

	log.Printf("[DuoAdmin] Request params: %+v", jsonParams)

	// Make the API call using v3 endpoint with JSONSignedCall (uses v5 signing)
	resp, body, err := c.JSONSignedCall(
		http.MethodPost,
		"/admin/v3/integrations",
		jsonParams,
		duoapi.UseTimeout,
	)
	if err != nil {
		log.Printf("[DuoAdmin] Failed to create OIDC integration: %v", err)
		return nil, fmt.Errorf("failed to create OIDC integration: %w", err)
	}

	log.Printf("[DuoAdmin] Create OIDC integration response status: %d", resp.StatusCode)
	log.Printf("[DuoAdmin] Create OIDC integration response body: %s", string(body))

	// Parse the response
	var result struct {
		Stat     string          `json:"stat"`
		Response OIDCIntegration `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[DuoAdmin] Failed to parse create OIDC integration response: %v", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if the API call was successful
	if result.Stat != "OK" {
		log.Printf("[DuoAdmin] Create OIDC integration failed. Stat: %s", result.Stat)
		return nil, fmt.Errorf("API returned error status: %s", result.Stat)
	}

	integration := result.Response
	log.Printf("[DuoAdmin] OIDC integration created successfully. Integration key: %s", integration.IntegrationKey)
	log.Printf("[DuoAdmin] Client ID: %s", integration.SSO.IDPMetadata.ClientID)
	log.Printf("[DuoAdmin] Discovery URL: %s", integration.SSO.IDPMetadata.DiscoveryURL)
	return &integration, nil
}
