package handlers

import (
	"fmt"
	"log"
	"user_experience_toolkit/internal/config"
	"user_experience_toolkit/internal/duoadmin"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ConfigHandler struct {
	Config *config.Config
}

func NewConfigHandler(cfg *config.Config) *ConfigHandler {
	return &ConfigHandler{Config: cfg}
}

// Show renders the configuration page with the table view
func (h *ConfigHandler) Show(c fiber.Ctx) error {
	// Get all tenants with their applications
	tenants := h.Config.GetAllTenants()

	// Build response with applications grouped by tenant
	type TenantWithApps struct {
		Tenant       config.Tenant
		Applications []config.Application
	}

	var tenantsWithApps []TenantWithApps
	for _, tenant := range tenants {
		apps := h.Config.GetApplicationsByTenant(tenant.ID)
		tenantsWithApps = append(tenantsWithApps, TenantWithApps{
			Tenant:       tenant,
			Applications: apps,
		})
	}

	return c.Render("configure", fiber.Map{
		"Tenants": tenantsWithApps,
	})
}

// ListApplications returns JSON list of all applications
func (h *ConfigHandler) ListApplications(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"applications": h.Config.Applications,
	})
}

// AddApplication adds a new application
func (h *ConfigHandler) AddApplication(c fiber.Ctx) error {
	var app config.Application

	if err := c.Bind().JSON(&app); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.Config.AddApplication(app); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":     "Application added successfully",
		"application": app,
	})
}

// UpdateApplication updates an existing application
func (h *ConfigHandler) UpdateApplication(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Application ID is required",
		})
	}

	var app config.Application
	if err := c.Bind().JSON(&app); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.Config.UpdateApplication(id, app); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":     "Application updated successfully",
		"application": app,
	})
}

// DeleteApplication deletes an application
func (h *ConfigHandler) DeleteApplication(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Application ID is required",
		})
	}

	if err := h.Config.DeleteApplication(id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Application deleted successfully",
	})
}

// AutoCreateApplicationRequest represents the request body for auto-creating an application
type AutoCreateApplicationRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // "websdk", "dmp", or "saml"
	Enabled  bool   `json:"enabled"`
	TenantID string `json:"tenant_id"` // Reference to tenant for Admin API creds
}

// AddTenantRequest represents the request body for adding a new tenant
type AddTenantRequest struct {
	Name           string `json:"name"`
	AdminAPIKey    string `json:"admin_api_key"`
	AdminAPISecret string `json:"admin_api_secret"`
	APIHostname    string `json:"api_hostname"`
}

// AutoCreateApplication creates a new application using the Duo Admin API
func (h *ConfigHandler) AutoCreateApplication(c fiber.Ctx) error {
	log.Printf("[ConfigHandler] AutoCreateApplication called")

	var req AutoCreateApplicationRequest

	if err := c.Bind().JSON(&req); err != nil {
		log.Printf("[ConfigHandler] Failed to parse request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	log.Printf("[ConfigHandler] Request received - Name: %s, Type: %s, Enabled: %v, TenantID: %s",
		req.Name, req.Type, req.Enabled, req.TenantID)

	// Validate required fields
	if req.Name == "" {
		log.Printf("[ConfigHandler] Validation failed: Application name is required")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Application name is required",
		})
	}
	if req.Type != "websdk" && req.Type != "dmp" && req.Type != "saml" && req.Type != "oidc" {
		log.Printf("[ConfigHandler] Validation failed: Invalid type '%s'", req.Type)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Type must be 'websdk', 'dmp', 'saml', or 'oidc'",
		})
	}
	if req.TenantID == "" {
		log.Printf("[ConfigHandler] Validation failed: Tenant ID is required")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Tenant ID is required",
		})
	}

	// Get the tenant to retrieve Admin API credentials
	tenant, err := h.Config.GetTenant(req.TenantID)
	if err != nil {
		log.Printf("[ConfigHandler] Failed to get tenant: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Tenant not found: " + err.Error(),
		})
	}

	log.Printf("[ConfigHandler] Using tenant '%s' with hostname: %s", tenant.Name, tenant.APIHostname)

	// Create Duo Admin API client with tenant's credentials
	adminClient := duoadmin.NewClient(tenant.AdminAPIKey, tenant.AdminAPISecret, tenant.APIHostname)

	// Validate credentials first
	log.Printf("[ConfigHandler] Validating Admin API credentials...")
	if err := adminClient.ValidateCredentials(); err != nil {
		log.Printf("[ConfigHandler] Credential validation failed: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Admin API credentials or insufficient permissions: " + err.Error(),
		})
	}

	log.Printf("[ConfigHandler] Credentials validated successfully")

	// Prefix application name with tenant name
	fullAppName := fmt.Sprintf("%s - %s", tenant.Name, req.Name)

	var app config.Application

	// Handle SAML type separately
	if req.Type == "saml" {
		log.Printf("[ConfigHandler] Creating SAML integration: %s", fullAppName)

		// For SAML, we need to generate app ID and URLs first, then create everything together
		baseURL := c.BaseURL()

		// Generate a new UUID for the app
		appID := uuid.New().String()

		// Generate SAML URLs using the new app ID
		entityID := fmt.Sprintf("%s/app/%s/saml", baseURL, appID)
		acsURL := fmt.Sprintf("%s/app/%s/saml/acs", baseURL, appID)
		metadataURL := fmt.Sprintf("%s/app/%s/saml/metadata", baseURL, appID)

		log.Printf("[ConfigHandler] Generated app ID: %s", appID)
		log.Printf("[ConfigHandler] Entity ID: %s", entityID)
		log.Printf("[ConfigHandler] ACS URL: %s", acsURL)

		// Create SAML integration via Admin API
		samlIntegration, err := adminClient.CreateSAMLIntegration(duoadmin.CreateSAMLIntegrationParams{
			Name:            fullAppName,
			EntityID:        entityID,
			ACSURL:          acsURL,
			NameIDFormat:    "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			NameIDAttribute: "<Email Address>",
		})
		if err != nil {
			log.Printf("[ConfigHandler] Failed to create SAML integration: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create SAML application via Duo Admin API: " + err.Error(),
			})
		}

		log.Printf("[ConfigHandler] SAML integration created successfully")

		// Extract IDP metadata from the API response
		integrationKey := samlIntegration.IntegrationKey
		idpEntityID := samlIntegration.SSO.IDPMetadata.EntityID
		idpSSOURL := samlIntegration.SSO.IDPMetadata.SSOURL
		idpCertificate := samlIntegration.SSO.IDPMetadata.Cert

		log.Printf("[ConfigHandler] Integration Key: %s", integrationKey)
		log.Printf("[ConfigHandler] IDP Entity ID: %s", idpEntityID)
		log.Printf("[ConfigHandler] IDP SSO URL: %s", idpSSOURL)
		log.Printf("[ConfigHandler] IDP Certificate length: %d bytes", len(idpCertificate))

		// Create the complete application object with all required fields
		app = config.Application{
			ID:          appID,
			TenantID:    req.TenantID,
			Name:        fullAppName,
			Type:        "saml",
			Enabled:     req.Enabled,
			ClientID:    integrationKey,
			APIHostname: tenant.APIHostname,
			EntityID:    entityID,
			ACSURL:      acsURL,
			MetadataURL: metadataURL,
			// IDP metadata from Duo API response
			IDPEntityID:    idpEntityID,
			IDPSSOURL:      idpSSOURL,
			IDPCertificate: idpCertificate,
		}

		// Now add the complete app to config (only once with all fields)
		if err := h.Config.AddApplication(app); err != nil {
			log.Printf("[ConfigHandler] Failed to save application to config: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		log.Printf("[ConfigHandler] SAML application created successfully. ID: %s", app.ID)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message":     "SAML application created successfully via Duo Admin API",
			"application": app,
		})
	}

	// Handle OIDC type separately
	if req.Type == "oidc" {
		log.Printf("[ConfigHandler] Creating OIDC integration: %s", fullAppName)

		// For OIDC, we need to generate app ID and redirect URI first, then create everything together
		baseURL := c.BaseURL()

		// Generate a new UUID for the app
		appID := uuid.New().String()

		// Generate OIDC redirect URI using the new app ID
		redirectURI := fmt.Sprintf("%s/app/%s/oidc/callback", baseURL, appID)

		log.Printf("[ConfigHandler] Generated app ID: %s", appID)
		log.Printf("[ConfigHandler] Redirect URI: %s", redirectURI)

		// Create OIDC integration via Admin API
		oidcIntegration, err := adminClient.CreateOIDCIntegration(duoadmin.CreateOIDCIntegrationParams{
			Name:                   fullAppName,
			RedirectURIs:           []string{redirectURI},
			Scopes:                 []string{}, // Only openid, which is added automatically
			AccessTokenLifespan:    3600,
			AllowPKCEOnly:          false,
			EnableRefreshToken:     true,
			RefreshTokenChainLife:  2592000,
			RefreshTokenSingleLife: 86400,
		})
		if err != nil {
			log.Printf("[ConfigHandler] Failed to create OIDC integration: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create OIDC application via Duo Admin API: " + err.Error(),
			})
		}

		log.Printf("[ConfigHandler] OIDC integration created successfully")

		// Extract IDP metadata from the API response
		clientID := oidcIntegration.SSO.IDPMetadata.ClientID
		clientSecret := oidcIntegration.SSO.IDPMetadata.ClientSecret
		discoveryURL := oidcIntegration.SSO.IDPMetadata.DiscoveryURL
		issuer := oidcIntegration.SSO.IDPMetadata.Issuer
		authEndpoint := oidcIntegration.SSO.IDPMetadata.AuthorizeEndpointURL
		tokenEndpoint := oidcIntegration.SSO.IDPMetadata.TokenEndpointURL
		userinfoEndpoint := oidcIntegration.SSO.IDPMetadata.UserInfoEndpointURL
		jwksEndpoint := oidcIntegration.SSO.IDPMetadata.JWKSEndpointURL

		log.Printf("[ConfigHandler] Client ID: %s", clientID)
		log.Printf("[ConfigHandler] Discovery URL: %s", discoveryURL)
		log.Printf("[ConfigHandler] Issuer: %s", issuer)

		// Create the complete application object with all required fields
		app = config.Application{
			ID:           appID,
			TenantID:     req.TenantID,
			Name:         fullAppName,
			Type:         "oidc",
			Enabled:      req.Enabled,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			APIHostname:  tenant.APIHostname,
			RedirectURI:  redirectURI,
			// IDP metadata from Duo API response
			IDPDiscoveryURL:          discoveryURL,
			IDPIssuer:                issuer,
			IDPAuthorizationEndpoint: authEndpoint,
			IDPTokenEndpoint:         tokenEndpoint,
			IDPUserInfoEndpoint:      userinfoEndpoint,
			IDPJWKSEndpoint:          jwksEndpoint,
		}

		// Now add the complete app to config (only once with all fields)
		if err := h.Config.AddApplication(app); err != nil {
			log.Printf("[ConfigHandler] Failed to save application to config: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		log.Printf("[ConfigHandler] OIDC application created successfully. ID: %s", app.ID)

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message":     "OIDC application created successfully via Duo Admin API",
			"application": app,
		})
	}

	// Map the type to Duo's integration type for websdk/dmp
	// For Web SDK v4 (Universal Prompt), the integration type is "websdk"
	// For Device Management Portal, the integration type is "device-management-portal"
	integrationType := "websdk" // Default to WebSDK
	if req.Type == "dmp" {
		integrationType = "device-management-portal"
	}

	log.Printf("[ConfigHandler] Creating integration with type: %s, name: %s", integrationType, fullAppName)

	// Create the integration via Admin API
	integration, err := adminClient.CreateIntegration(duoadmin.CreateIntegrationParams{
		Name:    fullAppName,
		Type:    integrationType,
		Enabled: req.Enabled,
	})
	if err != nil {
		log.Printf("[ConfigHandler] Failed to create integration: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create application via Duo Admin API: " + err.Error(),
		})
	}

	log.Printf("[ConfigHandler] Integration created successfully")

	// Create the application config with the returned credentials
	app = config.Application{
		TenantID:     req.TenantID,
		Name:         fullAppName,
		Type:         req.Type,
		Enabled:      req.Enabled,
		ClientID:     integration.IntegrationKey,
		ClientSecret: integration.SecretKey,
		APIHostname:  tenant.APIHostname,
	}

	log.Printf("[ConfigHandler] Saving application to config...")

	// Add to config
	if err := h.Config.AddApplication(app); err != nil {
		log.Printf("[ConfigHandler] Failed to save application to config: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Printf("[ConfigHandler] Application created and saved successfully. ID: %s", app.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":     "Application created successfully via Duo Admin API",
		"application": app,
	})
}

// Tenant Handlers

// ListTenants returns JSON list of all tenants with their applications
func (h *ConfigHandler) ListTenants(c fiber.Ctx) error {
	tenants := h.Config.GetAllTenants()

	// Build response with applications grouped by tenant
	type TenantWithApps struct {
		config.Tenant
		Applications []config.Application `json:"applications"`
	}

	var response []TenantWithApps
	for _, tenant := range tenants {
		apps := h.Config.GetApplicationsByTenant(tenant.ID)
		response = append(response, TenantWithApps{
			Tenant:       tenant,
			Applications: apps,
		})
	}

	return c.JSON(fiber.Map{
		"tenants": response,
	})
}

// AddTenant adds a new tenant
func (h *ConfigHandler) AddTenant(c fiber.Ctx) error {
	log.Printf("[ConfigHandler] AddTenant called")

	var req AddTenantRequest

	if err := c.Bind().JSON(&req); err != nil {
		log.Printf("[ConfigHandler] Failed to parse request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	log.Printf("[ConfigHandler] Request received - Name: %s, APIHostname: %s", req.Name, req.APIHostname)

	// Create Duo Admin API client to validate credentials
	adminClient := duoadmin.NewClient(req.AdminAPIKey, req.AdminAPISecret, req.APIHostname)

	// Validate credentials first
	log.Printf("[ConfigHandler] Validating Admin API credentials...")
	if err := adminClient.ValidateCredentials(); err != nil {
		log.Printf("[ConfigHandler] Credential validation failed: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Admin API credentials or insufficient permissions: " + err.Error(),
		})
	}

	log.Printf("[ConfigHandler] Credentials validated successfully")

	// Create tenant
	tenant := config.Tenant{
		Name:           req.Name,
		AdminAPIKey:    req.AdminAPIKey,
		AdminAPISecret: req.AdminAPISecret,
		APIHostname:    req.APIHostname,
	}

	if err := h.Config.AddTenant(tenant); err != nil {
		log.Printf("[ConfigHandler] Failed to add tenant: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Printf("[ConfigHandler] Tenant added successfully. ID: %s", tenant.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Tenant added successfully",
		"tenant":  tenant,
	})
}

// DeleteTenant deletes a tenant and all its applications
func (h *ConfigHandler) DeleteTenant(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Tenant ID is required",
		})
	}

	log.Printf("[ConfigHandler] Deleting tenant: %s", id)

	if err := h.Config.DeleteTenant(id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Printf("[ConfigHandler] Tenant and associated applications deleted successfully")

	return c.JSON(fiber.Map{
		"message": "Tenant and all associated applications deleted successfully",
	})
}
