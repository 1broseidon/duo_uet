package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"user_experience_toolkit/internal/config"
	"user_experience_toolkit/internal/duoadmin"

	"github.com/google/uuid"
)

func main() {
	// Flags (optional overrides)
	configPath := flag.String("config", "config.yaml", "Path to config.yaml")
	tenantName := flag.String("tenant", "Premier", "Tenant name to use")
	baseURL := flag.String("base-url", "http://localhost:8080", "Base URL for generating SAML endpoints")
	appSuffix := flag.String("name", "SAML Test", "Application name suffix")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Tenants) == 0 {
		log.Fatalf("No tenants found in %s", *configPath)
	}

	// Select tenant by name (case-insensitive), fallback to first
	var tenant *config.Tenant
	for i := range cfg.Tenants {
		if strings.EqualFold(cfg.Tenants[i].Name, *tenantName) {
			tenant = &cfg.Tenants[i]
			break
		}
	}
	if tenant == nil {
		log.Printf("Tenant '%s' not found, falling back to first tenant '%s'", *tenantName, cfg.Tenants[0].Name)
		tenant = &cfg.Tenants[0]
	}

	// Create Duo Admin API client
	client := duoadmin.NewClient(tenant.AdminAPIKey, tenant.AdminAPISecret, tenant.APIHostname)

	// Optional: validate credentials first
	if err := client.ValidateCredentials(); err != nil {
		log.Fatalf("Admin API credential validation failed: %v", err)
	}

	// Generate app-specific URLs
	appID := uuid.New().String()
	entityID := fmt.Sprintf("%s/app/%s/saml", *baseURL, appID)
	acsURL := fmt.Sprintf("%s/app/%s/saml/acs", *baseURL, appID)
	appName := fmt.Sprintf("%s - %s", tenant.Name, *appSuffix)

	log.Printf("Creating SAML integration on tenant '%s' (%s)", tenant.Name, tenant.APIHostname)
	log.Printf("EntityID: %s", entityID)
	log.Printf("ACS URL: %s", acsURL)

	res, err := client.CreateSAMLIntegration(duoadmin.CreateSAMLIntegrationParams{
		Name:            appName,
		EntityID:        entityID,
		ACSURL:          acsURL,
		NameIDFormat:    "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
		NameIDAttribute: "mail",
	})
	if err != nil {
		log.Fatalf("Failed to create SAML integration: %v", err)
	}

	// Output minimal details
	fmt.Fprintf(os.Stdout, "SAML integration created\n")
	fmt.Fprintf(os.Stdout, "Name: %s\n", res.Name)
	fmt.Fprintf(os.Stdout, "Integration Key: %s\n", res.IntegrationKey)
	fmt.Fprintf(os.Stdout, "EntityID: %s\n", entityID)
	fmt.Fprintf(os.Stdout, "ACS URL: %s\n", acsURL)
}
