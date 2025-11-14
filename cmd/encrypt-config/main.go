package main

import (
	"fmt"
	"log"
	"os"

	"user_experience_toolkit/internal/crypto"

	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: encrypt-config <config-file>")
		fmt.Println("\nEncrypts sensitive fields in a YAML configuration file.")
		fmt.Println("\nEnvironment variables:")
		fmt.Println("  UET_MASTER_KEY - Master encryption key (optional)")
		fmt.Println("                   If not set, uses/creates .uet_key file")
		os.Exit(1)
	}

	configPath := os.Args[1]

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	// Parse YAML into generic structure
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	// Create crypto manager
	cm, err := crypto.NewCryptoManager()
	if err != nil {
		log.Fatalf("Failed to create crypto manager: %v", err)
	}

	// Encrypt tenants
	if tenants, ok := yamlData["tenants"].([]interface{}); ok {
		for _, tenant := range tenants {
			if tenantMap, ok := tenant.(map[string]interface{}); ok {
				if err := cm.EncryptSensitiveFields(tenantMap); err != nil {
					log.Printf("Warning: Failed to encrypt tenant: %v", err)
				}
			}
		}
	}

	// Encrypt applications
	if apps, ok := yamlData["applications"].([]interface{}); ok {
		for _, app := range apps {
			if appMap, ok := app.(map[string]interface{}); ok {
				if err := cm.EncryptSensitiveFields(appMap); err != nil {
					log.Printf("Warning: Failed to encrypt application: %v", err)
				}
			}
		}
	}

	// Write back to file
	output, err := yaml.Marshal(yamlData)
	if err != nil {
		log.Fatalf("Failed to marshal YAML: %v", err)
	}

	if err := os.WriteFile(configPath, output, 0644); err != nil {
		log.Fatalf("Failed to write config: %v", err)
	}

	fmt.Printf("✅ Successfully encrypted secrets in %s\n", configPath)
	fmt.Println("\nEncrypted fields:")
	fmt.Println("  - admin_api_secret (in tenants)")
	fmt.Println("  - client_secret (in applications)")
	fmt.Println("  - signing_key (in applications)")
	fmt.Println("\nTo decrypt, use: decrypt-config", configPath)
}
