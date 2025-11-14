package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"user_experience_toolkit/internal/config"

	"github.com/duosecurity/duo_universal_golang/duouniversal"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

// getAdminHostname converts API hostname to Admin Panel hostname
func getAdminHostname(apiHostname string) string {
	return strings.Replace(apiHostname, "api-", "admin-", 1)
}

type DMPHandler struct {
	App       *config.Application
	DuoClient *duouniversal.Client
	Store     *session.Store
}

// NewDMPHandlerFromApp creates a new DMP handler from an Application config
func NewDMPHandlerFromApp(app *config.Application, store *session.Store, baseURL string) (*DMPHandler, error) {
	if app.GetApplicationType() != "dmp" {
		return nil, fmt.Errorf("application is not configured as DMP")
	}

	// Generate redirect URI based on application ID
	redirectURI := fmt.Sprintf("%s/app/%s/callback", baseURL, app.ID)

	duoClient, err := duouniversal.NewClient(
		app.ClientID,
		app.ClientSecret,
		app.APIHostname,
		redirectURI,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Duo client: %v", err)
	}

	return &DMPHandler{
		App:       app,
		DuoClient: duoClient,
		Store:     store,
	}, nil
}

func (h *DMPHandler) Login(c fiber.Ctx) error {
	return c.Render("login", fiber.Map{
		"AppType":        "dmp",
		"Message":        "",
		"AppName":        h.App.Name,
		"AppID":          h.App.ID,
		"APIHostname":    h.App.APIHostname,
		"AdminHostname":  getAdminHostname(h.App.APIHostname),
		"IntegrationKey": h.App.ClientID,
	})
}

func (h *DMPHandler) ProcessLogin(c fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	// Basic validation
	if username == "" || password == "" {
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "Incorrect username or password",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	// Check if Duo is configured
	if h.DuoClient == nil {
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "Duo is not configured properly.",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	// Perform health check
	_, err := h.DuoClient.HealthCheck()
	if err != nil {
		log.Printf("Duo health check failed: %v", err)
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "2FA Unavailable. Confirm Duo client/secret/host values are correct",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	// Generate state for CSRF protection
	state, err := h.DuoClient.GenerateState()
	if err != nil {
		log.Printf("Failed to generate state: %v", err)
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "Failed to generate authentication state",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	// Store state and username in session
	sess, err := h.Store.Get(c)
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "Session error",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	sess.Set("state", state)
	sess.Set("username", username)

	if err := sess.Save(); err != nil {
		log.Printf("Failed to save session: %v", err)
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "Failed to save session",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	// Generate auth URL and redirect
	authURL, err := h.DuoClient.CreateAuthURL(username, state)
	if err != nil {
		log.Printf("Failed to generate auth URL: %v", err)
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "Failed to generate authentication URL",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	return c.Redirect().To(authURL)
}

func (h *DMPHandler) Callback(c fiber.Ctx) error {
	// Check for errors from Duo
	if errMsg := c.Query("error"); errMsg != "" {
		errDesc := c.Query("error_description")
		log.Printf("Duo auth error: %s - %s", errMsg, errDesc)
		return c.SendString(fmt.Sprintf("Got Error: %s: %s", errMsg, errDesc))
	}

	// Get authorization code
	code := c.Query("duo_code")
	state := c.Query("state")

	if code == "" || state == "" {
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "Missing authorization code or state",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	// Retrieve session data
	sess, err := h.Store.Get(c)
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "Session error",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	savedState := sess.Get("state")
	username := sess.Get("username")

	if savedState == nil || username == nil {
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "No saved state, please login again",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	// Verify state matches
	if state != savedState.(string) {
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "Duo state does not match saved state",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	// Exchange code for token
	decodedToken, err := h.DuoClient.ExchangeAuthorizationCodeFor2faResult(code, username.(string))
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		return c.Render("login", fiber.Map{
			"AppType":        "dmp",
			"Message":        "Error decoding Duo result. Confirm device clock is correct.",
			"AppName":        h.App.Name,
			"AppID":          h.App.ID,
			"APIHostname":    h.App.APIHostname,
			"AdminHostname":  getAdminHostname(h.App.APIHostname),
			"IntegrationKey": h.App.ClientID,
		})
	}

	// Format token as JSON for display
	tokenJSON, err := json.MarshalIndent(decodedToken, "", "  ")
	if err != nil {
		tokenJSON = []byte(fmt.Sprintf("%+v", decodedToken))
	}

	// Clean up session
	sess.Delete("state")
	sess.Delete("username")
	sess.Save()

	return c.Render("success", fiber.Map{
		"AppType":        "dmp",
		"TokenData":      string(tokenJSON),
		"AppName":        h.App.Name,
		"AppID":          h.App.ID,
		"AdminHostname":  getAdminHostname(h.App.APIHostname),
		"IntegrationKey": h.App.ClientID,
	})
}
