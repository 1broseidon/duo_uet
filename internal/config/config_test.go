package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid config",
			content: `
tenants:
  - id: "test-tenant-1"
    name: "Test Tenant"
    admin_api_key: "testkey"
    admin_api_secret: "testsecret"
    api_hostname: "api-test.duosecurity.com"
applications:
  - id: "test-app-1"
    name: "Test Application"
    type: "websdk"
    enabled: true
    client_id: "test_client_id"
    client_secret: "test_client_secret"
    api_hostname: "api-test.duosecurity.com"
`,
			wantErr: false,
		},
		{
			name:    "invalid yaml",
			content: "invalid: [yaml content",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if cfg == nil {
					t.Error("LoadConfig() returned nil config")
					return
				}
				if cfg.filepath != configPath {
					t.Errorf("Config filepath = %v, want %v", cfg.filepath, configPath)
				}
			}
		})
	}
}

func TestLoadConfigNonexistent(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("LoadConfig() should return error for nonexistent file")
	}
}

func TestConfigBackwardCompatibility(t *testing.T) {
	content := `
applications:
  - id: "test-app-1"
    name: "Old DMP App"
    is_dmp: true
    enabled: true
    client_id: "test_client_id"
    client_secret: "test_client_secret"
    api_hostname: "api-test.duosecurity.com"
  - id: "test-app-2"
    name: "Old WebSDK App"
    is_dmp: false
    enabled: true
    client_id: "test_client_id2"
    client_secret: "test_client_secret2"
    api_hostname: "api-test.duosecurity.com"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Applications) != 2 {
		t.Fatalf("Expected 2 applications, got %d", len(cfg.Applications))
	}

	// Check first app was migrated to dmp
	if cfg.Applications[0].Type != "dmp" {
		t.Errorf("Application[0].Type = %v, want dmp", cfg.Applications[0].Type)
	}

	// Check second app was migrated to websdk
	if cfg.Applications[1].Type != "websdk" {
		t.Errorf("Application[1].Type = %v, want websdk", cfg.Applications[1].Type)
	}
}

func TestGetApplication(t *testing.T) {
	cfg := &Config{
		Applications: []Application{
			{ID: "app1", Name: "App 1"},
			{ID: "app2", Name: "App 2"},
		},
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"existing app", "app1", false},
		{"another existing app", "app2", false},
		{"nonexistent app", "app3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := cfg.GetApplication(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetApplication() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && app.ID != tt.id {
				t.Errorf("GetApplication() returned wrong app: got %v, want %v", app.ID, tt.id)
			}
		})
	}
}

func TestGetEnabledApplications(t *testing.T) {
	cfg := &Config{
		Applications: []Application{
			{ID: "app1", Name: "App 1", Enabled: true},
			{ID: "app2", Name: "App 2", Enabled: false},
			{ID: "app3", Name: "App 3", Enabled: true},
		},
	}

	enabled := cfg.GetEnabledApplications()
	if len(enabled) != 2 {
		t.Errorf("GetEnabledApplications() returned %d apps, want 2", len(enabled))
	}

	for _, app := range enabled {
		if !app.Enabled {
			t.Errorf("GetEnabledApplications() returned disabled app: %v", app.ID)
		}
	}
}

func TestAddApplication(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create initial config
	initialContent := `applications: []`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	tests := []struct {
		name    string
		app     Application
		wantErr bool
	}{
		{
			name: "valid websdk app",
			app: Application{
				Name:         "Test WebSDK",
				Type:         "websdk",
				Enabled:      true,
				ClientID:     "test_client",
				ClientSecret: "test_secret",
				APIHostname:  "api-test.duosecurity.com",
			},
			wantErr: false,
		},
		{
			name: "invalid app - missing name",
			app: Application{
				Type:         "websdk",
				Enabled:      true,
				ClientID:     "test_client",
				ClientSecret: "test_secret",
				APIHostname:  "api-test.duosecurity.com",
			},
			wantErr: true,
		},
		{
			name: "invalid app - missing client_id",
			app: Application{
				Name:         "Test App",
				Type:         "websdk",
				Enabled:      true,
				ClientSecret: "test_secret",
				APIHostname:  "api-test.duosecurity.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cfg.AddApplication(tt.app)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddApplication() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify the app was added
				found := false
				for _, app := range cfg.Applications {
					if app.Name == tt.app.Name {
						found = true
						// Verify ID was generated
						if app.ID == "" {
							t.Error("AddApplication() should generate ID if not provided")
						}
						break
					}
				}
				if !found {
					t.Error("AddApplication() did not add the application")
				}
			}
		})
	}
}

func TestUpdateApplication(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialContent := `
applications:
  - id: "app1"
    name: "Old Name"
    type: "websdk"
    enabled: true
    client_id: "old_client"
    client_secret: "old_secret"
    api_hostname: "api-old.duosecurity.com"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	updatedApp := Application{
		Name:         "New Name",
		Type:         "websdk",
		Enabled:      true,
		ClientID:     "new_client",
		ClientSecret: "new_secret",
		APIHostname:  "api-new.duosecurity.com",
	}

	err = cfg.UpdateApplication("app1", updatedApp)
	if err != nil {
		t.Errorf("UpdateApplication() error = %v", err)
	}

	// Verify update
	app, err := cfg.GetApplication("app1")
	if err != nil {
		t.Fatalf("GetApplication() error = %v", err)
	}
	if app.Name != "New Name" {
		t.Errorf("UpdateApplication() name = %v, want %v", app.Name, "New Name")
	}

	// Test updating nonexistent app
	err = cfg.UpdateApplication("nonexistent", updatedApp)
	if err == nil {
		t.Error("UpdateApplication() should return error for nonexistent app")
	}
}

func TestDeleteApplication(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialContent := `
applications:
  - id: "app1"
    name: "App 1"
    type: "websdk"
    enabled: true
    client_id: "client1"
    client_secret: "secret1"
    api_hostname: "api-test.duosecurity.com"
  - id: "app2"
    name: "App 2"
    type: "websdk"
    enabled: true
    client_id: "client2"
    client_secret: "secret2"
    api_hostname: "api-test.duosecurity.com"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Delete first app
	err = cfg.DeleteApplication("app1")
	if err != nil {
		t.Errorf("DeleteApplication() error = %v", err)
	}

	// Verify deletion
	if len(cfg.Applications) != 1 {
		t.Errorf("After deletion, expected 1 application, got %d", len(cfg.Applications))
	}
	if cfg.Applications[0].ID != "app2" {
		t.Errorf("Wrong application remained: %v", cfg.Applications[0].ID)
	}

	// Test deleting nonexistent app
	err = cfg.DeleteApplication("nonexistent")
	if err == nil {
		t.Error("DeleteApplication() should return error for nonexistent app")
	}
}

func TestApplicationGetApplicationType(t *testing.T) {
	tests := []struct {
		name string
		app  Application
		want string
	}{
		{
			name: "type set to dmp",
			app:  Application{Type: "dmp"},
			want: "dmp",
		},
		{
			name: "type set to websdk",
			app:  Application{Type: "websdk"},
			want: "websdk",
		},
		{
			name: "type set to saml",
			app:  Application{Type: "saml"},
			want: "saml",
		},
		{
			name: "legacy is_dmp true",
			app:  Application{IsDMP: true},
			want: "dmp",
		},
		{
			name: "legacy is_dmp false",
			app:  Application{IsDMP: false},
			want: "websdk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.app.GetApplicationType(); got != tt.want {
				t.Errorf("Application.GetApplicationType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigIsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "has enabled app",
			cfg: &Config{
				Applications: []Application{
					{Enabled: true},
				},
			},
			want: true,
		},
		{
			name: "no enabled apps",
			cfg: &Config{
				Applications: []Application{
					{Enabled: false},
				},
			},
			want: false,
		},
		{
			name: "empty config",
			cfg:  &Config{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsConfigured(); got != tt.want {
				t.Errorf("Config.IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenantManagement(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialContent := `tenants: []`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Test AddTenant
	tenant := Tenant{
		Name:           "Test Tenant",
		AdminAPIKey:    "test_key",
		AdminAPISecret: "test_secret",
		APIHostname:    "api-test.duosecurity.com",
	}

	err = cfg.AddTenant(tenant)
	if err != nil {
		t.Errorf("AddTenant() error = %v", err)
	}

	// Verify tenant was added
	tenants := cfg.GetAllTenants()
	if len(tenants) != 1 {
		t.Fatalf("Expected 1 tenant, got %d", len(tenants))
	}

	tenantID := tenants[0].ID

	// Test GetTenant
	retrievedTenant, err := cfg.GetTenant(tenantID)
	if err != nil {
		t.Errorf("GetTenant() error = %v", err)
	}
	if retrievedTenant.Name != "Test Tenant" {
		t.Errorf("GetTenant() name = %v, want %v", retrievedTenant.Name, "Test Tenant")
	}

	// Test GetTenantByHostname
	retrievedTenant, err = cfg.GetTenantByHostname("api-test.duosecurity.com")
	if err != nil {
		t.Errorf("GetTenantByHostname() error = %v", err)
	}
	if retrievedTenant.ID != tenantID {
		t.Errorf("GetTenantByHostname() returned wrong tenant")
	}

	// Test DeleteTenant
	err = cfg.DeleteTenant(tenantID)
	if err != nil {
		t.Errorf("DeleteTenant() error = %v", err)
	}

	tenants = cfg.GetAllTenants()
	if len(tenants) != 0 {
		t.Errorf("After deletion, expected 0 tenants, got %d", len(tenants))
	}
}

func TestDeleteTenantWithApplications(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	tenantID := uuid.New().String()
	initialContent := `
tenants:
  - id: "` + tenantID + `"
    name: "Test Tenant"
    admin_api_key: "test_key"
    admin_api_secret: "test_secret"
    api_hostname: "api-test.duosecurity.com"
applications:
  - id: "app1"
    tenant_id: "` + tenantID + `"
    name: "App 1"
    type: "websdk"
    enabled: true
    client_id: "client1"
    client_secret: "secret1"
    api_hostname: "api-test.duosecurity.com"
  - id: "app2"
    tenant_id: "other-tenant"
    name: "App 2"
    type: "websdk"
    enabled: true
    client_id: "client2"
    client_secret: "secret2"
    api_hostname: "api-test.duosecurity.com"
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Delete tenant should also delete its applications
	err = cfg.DeleteTenant(tenantID)
	if err != nil {
		t.Errorf("DeleteTenant() error = %v", err)
	}

	// Verify tenant is deleted
	tenants := cfg.GetAllTenants()
	if len(tenants) != 0 {
		t.Errorf("After deletion, expected 0 tenants, got %d", len(tenants))
	}

	// Verify only the tenant's app was deleted, not others
	if len(cfg.Applications) != 1 {
		t.Errorf("Expected 1 application to remain, got %d", len(cfg.Applications))
	}
	if cfg.Applications[0].ID != "app2" {
		t.Errorf("Wrong application remained: %v", cfg.Applications[0].ID)
	}
}

func TestGetApplicationsByTenant(t *testing.T) {
	cfg := &Config{
		Applications: []Application{
			{ID: "app1", TenantID: "tenant1"},
			{ID: "app2", TenantID: "tenant2"},
			{ID: "app3", TenantID: "tenant1"},
		},
	}

	apps := cfg.GetApplicationsByTenant("tenant1")
	if len(apps) != 2 {
		t.Errorf("GetApplicationsByTenant() returned %d apps, want 2", len(apps))
	}

	apps = cfg.GetApplicationsByTenant("tenant2")
	if len(apps) != 1 {
		t.Errorf("GetApplicationsByTenant() returned %d apps, want 1", len(apps))
	}

	apps = cfg.GetApplicationsByTenant("nonexistent")
	if len(apps) != 0 {
		t.Errorf("GetApplicationsByTenant() for nonexistent tenant returned %d apps, want 0", len(apps))
	}
}

func TestValidateApplication(t *testing.T) {
	tests := []struct {
		name    string
		app     *Application
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid websdk app",
			app: &Application{
				Name:         "Test",
				Type:         "websdk",
				ClientID:     "test",
				ClientSecret: "test",
				APIHostname:  "api-test.duosecurity.com",
			},
			wantErr: false,
		},
		{
			name: "valid saml app",
			app: &Application{
				Name:        "Test SAML",
				Type:        "saml",
				EntityID:    "http://example.com",
				ACSURL:      "http://example.com/acs",
				APIHostname: "api-test.duosecurity.com",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			app: &Application{
				Type:         "websdk",
				ClientID:     "test",
				ClientSecret: "test",
				APIHostname:  "api-test.duosecurity.com",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			app: &Application{
				Name:         "Test",
				Type:         "invalid",
				ClientID:     "test",
				ClientSecret: "test",
				APIHostname:  "api-test.duosecurity.com",
			},
			wantErr: true,
		},
		{
			name: "saml missing entity_id",
			app: &Application{
				Name:        "Test SAML",
				Type:        "saml",
				ACSURL:      "http://example.com/acs",
				APIHostname: "api-test.duosecurity.com",
			},
			wantErr: true,
		},
		{
			name: "saml missing acs_url",
			app: &Application{
				Name:        "Test SAML",
				Type:        "saml",
				EntityID:    "http://example.com",
				APIHostname: "api-test.duosecurity.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateApplication(tt.app)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateApplication() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTenant(t *testing.T) {
	tests := []struct {
		name    string
		tenant  *Tenant
		wantErr bool
	}{
		{
			name: "valid tenant",
			tenant: &Tenant{
				Name:           "Test",
				AdminAPIKey:    "test_key",
				AdminAPISecret: "test_secret",
				APIHostname:    "api-test.duosecurity.com",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			tenant: &Tenant{
				AdminAPIKey:    "test_key",
				AdminAPISecret: "test_secret",
				APIHostname:    "api-test.duosecurity.com",
			},
			wantErr: true,
		},
		{
			name: "missing api key",
			tenant: &Tenant{
				Name:           "Test",
				AdminAPISecret: "test_secret",
				APIHostname:    "api-test.duosecurity.com",
			},
			wantErr: true,
		},
		{
			name: "missing api secret",
			tenant: &Tenant{
				Name:        "Test",
				AdminAPIKey: "test_key",
				APIHostname: "api-test.duosecurity.com",
			},
			wantErr: true,
		},
		{
			name: "missing hostname",
			tenant: &Tenant{
				Name:           "Test",
				AdminAPIKey:    "test_key",
				AdminAPISecret: "test_secret",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTenant(tt.tenant)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTenant() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
