package main

import (
	"embed"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"user_experience_toolkit/internal/config"
	"user_experience_toolkit/internal/handlers"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

//go:embed static
var staticFS embed.FS

//go:embed templates
var templatesFS embed.FS

const (
	defaultConfigPath = "/app/config/config.yaml"
	port              = ":8080"
)

func main() {
	// Get config path from environment or use default
	configPath := os.Getenv("UET_CONFIG_PATH")
	if configPath == "" {
		// Check if running in Docker (check for /app directory)
		if _, err := os.Stat("/app"); err == nil {
			configPath = defaultConfigPath
		} else {
			// Running locally, use current directory
			configPath = "config.yaml"
		}
	}

	log.Printf("Using config file: %s", configPath)

	// Load configuration (will auto-create if missing)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		Views: &templateEngine{},
	})

	// Setup session store
	store := session.NewStore()

	// Setup static files from embedded filesystem
	app.Get("/static/*", func(c fiber.Ctx) error {
		// Get the requested file path
		filePath := "static/" + c.Params("*")

		// Read the file from embedded FS
		data, err := staticFS.ReadFile(filePath)
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("File not found")
		}

		// Set content type based on file extension
		c.Set("Content-Type", getContentType(filePath))
		return c.Send(data)
	})

	// Initialize handlers
	homeHandler := handlers.NewHomeHandler(cfg)
	configHandler := handlers.NewConfigHandler(cfg)

	// Routes
	app.Get("/", homeHandler.Index)

	// Configuration routes
	app.Get("/configure", configHandler.Show)

	// API routes for configuration management
	app.Get("/api/config/applications", configHandler.ListApplications)
	app.Post("/api/config/applications", configHandler.AddApplication)
	app.Post("/api/config/applications/auto-create", configHandler.AutoCreateApplication)
	app.Put("/api/config/applications/:id", configHandler.UpdateApplication)
	app.Delete("/api/config/applications/:id", configHandler.DeleteApplication)

	// API routes for tenant management
	app.Get("/api/config/tenants", configHandler.ListTenants)
	app.Post("/api/config/tenants", configHandler.AddTenant)
	app.Delete("/api/config/tenants/:id", configHandler.DeleteTenant)

	// Dynamic application routes
	app.All("/app/:id/*", func(c fiber.Ctx) error {
		appID := c.Params("id")
		path := c.Params("*")

		// Get the application configuration
		app, err := cfg.GetApplication(appID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Application not found")
		}

		// Check if application is enabled
		if !app.Enabled {
			return c.Status(fiber.StatusForbidden).SendString("Application is disabled")
		}

		// Route based on application type
		appType := app.GetApplicationType()
		switch appType {
		case "dmp":
			return handleDMPRequest(c, app, path, store)
		case "saml":
			return handleSAMLRequest(c, app, path, store)
		case "oidc":
			return handleOIDCRequest(c, app, path, store)
		case "websdk":
			return handleV4Request(c, app, path, store)
		default:
			return handleV4Request(c, app, path, store)
		}
	})

	// Start server
	log.Printf("Server starting on http://localhost%s", port)
	log.Fatal(app.Listen(port))
}

// handleV4Request handles requests for V4 applications
func handleV4Request(c fiber.Ctx, app *config.Application, path string, store *session.Store) error {
	// Get base URL for redirect URI generation
	baseURL := c.BaseURL()

	handler, err := handlers.NewV4HandlerFromApp(app, store, baseURL)
	if err != nil {
		log.Printf("Failed to create V4 handler: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to initialize V4 handler")
	}

	switch {
	case path == "" || path == "/":
		if c.Method() == "GET" {
			return handler.Login(c)
		} else if c.Method() == "POST" {
			return handler.ProcessLogin(c)
		}
	case path == "callback":
		return handler.Callback(c)
	}

	return c.Status(fiber.StatusNotFound).SendString("Not found")
}

// handleDMPRequest handles requests for DMP applications
func handleDMPRequest(c fiber.Ctx, app *config.Application, path string, store *session.Store) error {
	// Get base URL for redirect URI generation
	baseURL := c.BaseURL()

	handler, err := handlers.NewDMPHandlerFromApp(app, store, baseURL)
	if err != nil {
		log.Printf("Failed to create DMP handler: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to initialize DMP handler")
	}

	switch {
	case path == "" || path == "/":
		if c.Method() == "GET" {
			return handler.Login(c)
		} else if c.Method() == "POST" {
			return handler.ProcessLogin(c)
		}
	case path == "callback":
		return handler.Callback(c)
	}

	return c.Status(fiber.StatusNotFound).SendString("Not found")
}

// handleSAMLRequest handles requests for SAML applications
func handleSAMLRequest(c fiber.Ctx, app *config.Application, path string, store *session.Store) error {
	// Get base URL for redirect URI generation
	baseURL := c.BaseURL()

	handler, err := handlers.NewSAMLHandlerFromApp(app, store, baseURL)
	if err != nil {
		log.Printf("Failed to create SAML handler: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to initialize SAML handler")
	}

	switch {
	case path == "" || path == "/" || path == "saml" || path == "saml/":
		return handler.Login(c)
	case path == "saml/initiate":
		return handler.InitiateSAML(c)
	case path == "saml/acs":
		if c.Method() == "POST" {
			return handler.ACS(c)
		}
	case path == "saml/metadata":
		return handler.Metadata(c)
	case path == "saml/slo":
		return handler.SLO(c)
	case path == "saml/success":
		return handler.Success(c)
	}

	return c.Status(fiber.StatusNotFound).SendString("Not found")
}

// handleOIDCRequest handles requests for OIDC applications
func handleOIDCRequest(c fiber.Ctx, app *config.Application, path string, store *session.Store) error {
	// Get base URL for redirect URI generation
	baseURL := c.BaseURL()

	handler, err := handlers.NewOIDCHandlerFromApp(app, store, baseURL)
	if err != nil {
		log.Printf("Failed to create OIDC handler: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to initialize OIDC handler")
	}

	switch {
	case path == "" || path == "/" || path == "oidc" || path == "oidc/":
		return handler.Login(c)
	case path == "oidc/initiate":
		return handler.InitiateOIDC(c)
	case path == "oidc/callback":
		return handler.Callback(c)
	case path == "oidc/success":
		return handler.Success(c)
	case path == "oidc/logout":
		return handler.Logout(c)
	}

	return c.Status(fiber.StatusNotFound).SendString("Not found")
}

// Custom template engine using html/template
type templateEngine struct{}

func (e *templateEngine) Load() error {
	return nil
}

func (e *templateEngine) Render(w io.Writer, name string, bind any, layout ...string) error {
	// Parse layout template from embedded filesystem
	layoutTmpl, err := template.ParseFS(templatesFS, "templates/layout.html")
	if err != nil {
		return err
	}

	// Parse content template from embedded filesystem
	contentTmpl, err := template.ParseFS(templatesFS, "templates/"+name+".html")
	if err != nil {
		return err
	}

	// Execute content template to get the rendered content
	var contentBuf strings.Builder
	if err := contentTmpl.Execute(&contentBuf, bind); err != nil {
		return err
	}

	// Create a map with the embedded content
	data := map[string]interface{}{
		"embed": template.HTML(contentBuf.String()),
	}

	// Execute layout template with embedded content
	return layoutTmpl.Execute(w, data)
}

// getContentType returns the MIME type based on file extension
func getContentType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".html":
		return "text/html"
	case ".json":
		return "application/json"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	default:
		return "application/octet-stream"
	}
}
