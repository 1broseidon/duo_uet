package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Tenant represents a Duo tenant with Admin API credentials
type Tenant struct {
	ID             string `yaml:"id" json:"id"`
	Name           string `yaml:"name" json:"name"`
	AdminAPIKey    string `yaml:"admin_api_key" json:"admin_api_key"`
	AdminAPISecret string `yaml:"admin_api_secret" json:"admin_api_secret"`
	APIHostname    string `yaml:"api_hostname" json:"api_hostname"`
}

// Application represents a single Duo application configuration
type Application struct {
	ID           string `yaml:"id" json:"id"`
	TenantID     string `yaml:"tenant_id,omitempty" json:"tenant_id,omitempty"` // Optional: references tenant
	Name         string `yaml:"name" json:"name"`
	Type         string `yaml:"type" json:"type"`                         // "websdk", "dmp", "saml", "oidc"
	IsDMP        bool   `yaml:"is_dmp,omitempty" json:"is_dmp,omitempty"` // Deprecated: for backward compatibility only
	Enabled      bool   `yaml:"enabled" json:"enabled"`
	ClientID     string `yaml:"client_id" json:"client_id"`
	ClientSecret string `yaml:"client_secret" json:"client_secret"`
	APIHostname  string `yaml:"api_hostname" json:"api_hostname"`

	// SAML-specific fields (Service Provider)
	EntityID    string `yaml:"entity_id,omitempty" json:"entity_id,omitempty"`
	ACSURL      string `yaml:"acs_url,omitempty" json:"acs_url,omitempty"`
	MetadataURL string `yaml:"metadata_url,omitempty" json:"metadata_url,omitempty"`
	SigningCert string `yaml:"signing_cert,omitempty" json:"signing_cert,omitempty"`
	SigningKey  string `yaml:"signing_key,omitempty" json:"signing_key,omitempty"`

	// SAML IDP metadata fields (Duo as Identity Provider)
	IDPEntityID    string `yaml:"idp_entity_id,omitempty" json:"idp_entity_id,omitempty"`
	IDPSSOURL      string `yaml:"idp_sso_url,omitempty" json:"idp_sso_url,omitempty"`
	IDPCertificate string `yaml:"idp_certificate,omitempty" json:"idp_certificate,omitempty"`

	// OIDC-specific fields (Relying Party)
	RedirectURI               string `yaml:"redirect_uri,omitempty" json:"redirect_uri,omitempty"`
	IDPDiscoveryURL           string `yaml:"idp_discovery_url,omitempty" json:"idp_discovery_url,omitempty"`
	IDPIssuer                 string `yaml:"idp_issuer,omitempty" json:"idp_issuer,omitempty"`
	IDPAuthorizationEndpoint  string `yaml:"idp_authorization_endpoint,omitempty" json:"idp_authorization_endpoint,omitempty"`
	IDPTokenEndpoint          string `yaml:"idp_token_endpoint,omitempty" json:"idp_token_endpoint,omitempty"`
	IDPUserInfoEndpoint       string `yaml:"idp_userinfo_endpoint,omitempty" json:"idp_userinfo_endpoint,omitempty"`
	IDPJWKSEndpoint           string `yaml:"idp_jwks_endpoint,omitempty" json:"idp_jwks_endpoint,omitempty"`
}

// Config represents the entire configuration file
type Config struct {
	Tenants      []Tenant      `yaml:"tenants,omitempty" json:"tenants,omitempty"`
	Applications []Application `yaml:"applications" json:"applications"`
	mu           sync.RWMutex  `yaml:"-" json:"-"`
	filepath     string        `yaml:"-" json:"-"`
}

// LoadConfig loads and parses the YAML configuration file
func LoadConfig(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	config := &Config{
		filepath: filepath,
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Migrate old IsDMP field to Type field for backward compatibility
	for i := range config.Applications {
		if config.Applications[i].Type == "" {
			if config.Applications[i].IsDMP {
				config.Applications[i].Type = "dmp"
			} else {
				config.Applications[i].Type = "websdk"
			}
		}
	}

	return config, nil
}

// Save writes the configuration back to the file
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	err = os.WriteFile(c.filepath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// GetApplication retrieves an application by ID
func (c *Config) GetApplication(id string) (*Application, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := range c.Applications {
		if c.Applications[i].ID == id {
			return &c.Applications[i], nil
		}
	}

	return nil, fmt.Errorf("application with id '%s' not found", id)
}

// GetEnabledApplications returns all enabled applications
func (c *Config) GetEnabledApplications() []Application {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var enabled []Application
	for _, app := range c.Applications {
		if app.Enabled {
			enabled = append(enabled, app)
		}
	}

	return enabled
}

// AddApplication adds a new application to the configuration
func (c *Config) AddApplication(app Application) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate ID if not provided
	if app.ID == "" {
		app.ID = uuid.New().String()
	}

	// Validate the application
	if err := validateApplication(&app); err != nil {
		return err
	}

	// Check for duplicate ID
	for _, existing := range c.Applications {
		if existing.ID == app.ID {
			return fmt.Errorf("application with id '%s' already exists", app.ID)
		}
	}

	c.Applications = append(c.Applications, app)
	return c.save()
}

// UpdateApplication updates an existing application
func (c *Config) UpdateApplication(id string, updatedApp Application) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate the application
	if err := validateApplication(&updatedApp); err != nil {
		return err
	}

	for i := range c.Applications {
		if c.Applications[i].ID == id {
			// Preserve the original ID
			updatedApp.ID = id
			c.Applications[i] = updatedApp
			return c.save()
		}
	}

	return fmt.Errorf("application with id '%s' not found", id)
}

// DeleteApplication removes an application from the configuration
func (c *Config) DeleteApplication(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.Applications {
		if c.Applications[i].ID == id {
			c.Applications = append(c.Applications[:i], c.Applications[i+1:]...)
			return c.save()
		}
	}

	return fmt.Errorf("application with id '%s' not found", id)
}

// save is an internal method that saves without locking (assumes lock is held)
func (c *Config) save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	err = os.WriteFile(c.filepath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// validateApplication validates an application configuration
func validateApplication(app *Application) error {
	if app.Name == "" {
		return fmt.Errorf("application name is required")
	}

	// Validate type
	validTypes := []string{"websdk", "dmp", "saml", "oidc"}
	if app.Type == "" {
		return fmt.Errorf("application type is required")
	}
	isValidType := false
	for _, validType := range validTypes {
		if app.Type == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return fmt.Errorf("invalid application type: %s (must be one of: websdk, dmp, saml, oidc)", app.Type)
	}

	// Type-specific validation
	if app.Type == "saml" {
		if app.EntityID == "" {
			return fmt.Errorf("entity_id is required for SAML applications")
		}
		if app.ACSURL == "" {
			return fmt.Errorf("acs_url is required for SAML applications")
		}
		// ClientID and ClientSecret are not required for SAML
	} else if app.Type == "oidc" {
		// For OIDC - require client credentials and redirect URI
		if app.ClientID == "" {
			return fmt.Errorf("client_id is required for OIDC applications")
		}
		if app.ClientSecret == "" {
			return fmt.Errorf("client_secret is required for OIDC applications")
		}
		if app.RedirectURI == "" {
			return fmt.Errorf("redirect_uri is required for OIDC applications")
		}
	} else {
		// For websdk, dmp - require client credentials
		if app.ClientID == "" {
			return fmt.Errorf("client_id is required")
		}
		if app.ClientSecret == "" {
			return fmt.Errorf("client_secret is required")
		}
	}

	if app.APIHostname == "" {
		return fmt.Errorf("api_hostname is required")
	}

	return nil
}

// GetApplicationType returns the application type (websdk, dmp, saml, oidc)
func (a *Application) GetApplicationType() string {
	if a.Type != "" {
		return a.Type
	}
	// Fallback for backward compatibility
	if a.IsDMP {
		return "dmp"
	}
	return "websdk"
}

// IsConfigured checks if the configuration has at least one enabled application
func (c *Config) IsConfigured() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, app := range c.Applications {
		if app.Enabled {
			return true
		}
	}

	return false
}

// Tenant Management Methods

// AddTenant adds a new tenant to the configuration
func (c *Config) AddTenant(tenant Tenant) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate ID if not provided
	if tenant.ID == "" {
		tenant.ID = uuid.New().String()
	}

	// Validate the tenant
	if err := validateTenant(&tenant); err != nil {
		return err
	}

	// Check for duplicate ID
	for _, existing := range c.Tenants {
		if existing.ID == tenant.ID {
			return fmt.Errorf("tenant with id '%s' already exists", tenant.ID)
		}
	}

	c.Tenants = append(c.Tenants, tenant)
	return c.save()
}

// GetTenant retrieves a tenant by ID
func (c *Config) GetTenant(id string) (*Tenant, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := range c.Tenants {
		if c.Tenants[i].ID == id {
			return &c.Tenants[i], nil
		}
	}

	return nil, fmt.Errorf("tenant with id '%s' not found", id)
}

// GetAllTenants returns all tenants
func (c *Config) GetAllTenants() []Tenant {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tenants := make([]Tenant, len(c.Tenants))
	copy(tenants, c.Tenants)
	return tenants
}

// GetTenantByHostname retrieves a tenant by API hostname
func (c *Config) GetTenantByHostname(hostname string) (*Tenant, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := range c.Tenants {
		if c.Tenants[i].APIHostname == hostname {
			return &c.Tenants[i], nil
		}
	}

	return nil, fmt.Errorf("tenant with hostname '%s' not found", hostname)
}

// DeleteTenant removes a tenant and all its associated applications
func (c *Config) DeleteTenant(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find tenant
	tenantIndex := -1
	for i := range c.Tenants {
		if c.Tenants[i].ID == id {
			tenantIndex = i
			break
		}
	}

	if tenantIndex == -1 {
		return fmt.Errorf("tenant with id '%s' not found", id)
	}

	// Delete all applications associated with this tenant
	var remainingApps []Application
	for _, app := range c.Applications {
		if app.TenantID != id {
			remainingApps = append(remainingApps, app)
		}
	}
	c.Applications = remainingApps

	// Delete the tenant
	c.Tenants = append(c.Tenants[:tenantIndex], c.Tenants[tenantIndex+1:]...)

	return c.save()
}

// GetApplicationsByTenant returns all applications for a specific tenant
func (c *Config) GetApplicationsByTenant(tenantID string) []Application {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var apps []Application
	for _, app := range c.Applications {
		if app.TenantID == tenantID {
			apps = append(apps, app)
		}
	}

	return apps
}

// validateTenant validates a tenant configuration
func validateTenant(tenant *Tenant) error {
	if tenant.Name == "" {
		return fmt.Errorf("tenant name is required")
	}

	if tenant.AdminAPIKey == "" {
		return fmt.Errorf("admin_api_key is required")
	}

	if tenant.AdminAPISecret == "" {
		return fmt.Errorf("admin_api_secret is required")
	}

	if tenant.APIHostname == "" {
		return fmt.Errorf("api_hostname is required")
	}

	return nil
}
