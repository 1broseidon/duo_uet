package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"user_experience_toolkit/internal/config"

	"github.com/duosecurity/duo_universal_golang/duouniversal"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

type V4Handler struct {
	App       *config.Application
	DuoClient *duouniversal.Client
	Store     *session.Store
}

// NewV4HandlerFromApp creates a new V4 handler from an Application config
func NewV4HandlerFromApp(app *config.Application, store *session.Store, baseURL string) (*V4Handler, error) {
	appType := app.GetApplicationType()
	if appType != "websdk" && appType != "" {
		return nil, fmt.Errorf("application is configured as %s, not WebSDK V4", appType)
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

	return &V4Handler{
		App:       app,
		DuoClient: duoClient,
		Store:     store,
	}, nil
}

func (h *V4Handler) Login(c fiber.Ctx) error {
	return c.Render("login", fiber.Map{
		"AppType": "v4",
		"Message": "",
		"AppName": h.App.Name,
		"AppID":   h.App.ID,
	})
}

func (h *V4Handler) ProcessLogin(c fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	// Basic validation
	if username == "" || password == "" {
		return c.Render("login", fiber.Map{
			"AppType": "v4",
			"Message": "Incorrect username or password",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	// Check if Duo is configured
	if h.DuoClient == nil {
		return c.Render("login", fiber.Map{
			"AppType": "v4",
			"Message": "Duo is not configured properly.",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	// Perform health check
	_, err := h.DuoClient.HealthCheck()
	if err != nil {
		log.Printf("Duo health check failed: %v", err)
		return c.Render("login", fiber.Map{
			"AppType": "v4",
			"Message": "2FA Unavailable. Confirm Duo client/secret/host values are correct",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	// Generate state for CSRF protection
	state, err := h.DuoClient.GenerateState()
	if err != nil {
		log.Printf("Failed to generate state: %v", err)
		return c.Render("login", fiber.Map{
			"AppType": "v4",
			"Message": "Failed to generate authentication state",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	// Store state and username in session
	sess, err := h.Store.Get(c)
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		return c.Render("login", fiber.Map{
			"AppType": "v4",
			"Message": "Session error",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	sess.Set("state", state)
	sess.Set("username", username)

	if err := sess.Save(); err != nil {
		log.Printf("Failed to save session: %v", err)
		return c.Render("login", fiber.Map{
			"AppType": "v4",
			"Message": "Failed to save session",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	// Generate auth URL and redirect
	authURL, err := h.DuoClient.CreateAuthURL(username, state)
	if err != nil {
		log.Printf("Failed to generate auth URL: %v", err)
		return c.Render("login", fiber.Map{
			"AppType": "v4",
			"Message": "Failed to generate authentication URL",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	return c.Redirect().To(authURL)
}

func (h *V4Handler) Callback(c fiber.Ctx) error {
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
			"AppType": "v4",
			"Message": "Missing authorization code or state",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	// Retrieve session data
	sess, err := h.Store.Get(c)
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		return c.Render("login", fiber.Map{
			"AppType": "v4",
			"Message": "Session error",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	savedState := sess.Get("state")
	username := sess.Get("username")

	if savedState == nil || username == nil {
		return c.Render("login", fiber.Map{
			"AppType": "v4",
			"Message": "No saved state, please login again",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	// Verify state matches
	if state != savedState.(string) {
		return c.Render("login", fiber.Map{
			"AppType": "v4",
			"Message": "Duo state does not match saved state",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	// Exchange code for token
	decodedToken, err := h.DuoClient.ExchangeAuthorizationCodeFor2faResult(code, username.(string))
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		return c.Render("login", fiber.Map{
			"AppType": "v4",
			"Message": "Error decoding Duo result. Confirm device clock is correct.",
			"AppName": h.App.Name,
			"AppID":   h.App.ID,
		})
	}

	// Format token as JSON for display
	tokenJSON, err := json.MarshalIndent(decodedToken, "", "  ")
	if err != nil {
		tokenJSON = []byte(fmt.Sprintf("%+v", decodedToken))
	}

	// Extract key fields for display from the TokenResponse struct
	userEmail := decodedToken.AuthContext.Email
	if userEmail == "" {
		userEmail = decodedToken.PreferredUsername
	}

	authResult := decodedToken.AuthResult.Result
	authStatus := decodedToken.AuthResult.StatusMsg
	authFactor := decodedToken.AuthContext.Factor

	// Build device string from access device info
	authDevice := ""
	if decodedToken.AuthContext.AccessDevice.Browser != "" {
		authDevice = decodedToken.AuthContext.AccessDevice.Browser
		if decodedToken.AuthContext.AccessDevice.BrowserVersion != "" {
			authDevice += " " + decodedToken.AuthContext.AccessDevice.BrowserVersion
		}
	}
	if decodedToken.AuthContext.AccessDevice.Os != "" {
		if authDevice != "" {
			authDevice += " on "
		}
		authDevice += decodedToken.AuthContext.AccessDevice.Os
	}

	// Build location string
	authLocation := ""
	loc := decodedToken.AuthContext.AccessDevice.Location
	if loc.City != "" && loc.State != "" {
		authLocation = fmt.Sprintf("%s, %s", loc.City, loc.State)
	} else if loc.Country != "" {
		authLocation = loc.Country
	}

	// Clean up session
	sess.Delete("state")
	sess.Delete("username")
	sess.Save()

	return c.Render("success", fiber.Map{
		"AppType":      "v4",
		"TokenData":    string(tokenJSON),
		"AppName":      h.App.Name,
		"AppID":        h.App.ID,
		"AuthResult":   authResult,
		"AuthStatus":   authStatus,
		"AuthFactor":   authFactor,
		"UserEmail":    userEmail,
		"AuthDevice":   authDevice,
		"AuthLocation": authLocation,
	})
}
