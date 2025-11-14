package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"user_experience_toolkit/internal/config"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"golang.org/x/oauth2"
)

type OIDCHandler struct {
	App          *config.Application
	Session      *session.Store
	BaseURL      string
	Provider     *oidc.Provider
	OAuth2Config oauth2.Config
	Verifier     *oidc.IDTokenVerifier
}

// NewOIDCHandlerFromApp creates an OIDC handler from an application configuration
func NewOIDCHandlerFromApp(app *config.Application, store *session.Store, baseURL string) (*OIDCHandler, error) {
	log.Printf("[OIDCHandler] Initializing OIDC handler for app: %s (ID: %s)", app.Name, app.ID)

	ctx := context.Background()

	// Determine the issuer URL (not the full discovery URL)
	// oidc.NewProvider automatically appends /.well-known/openid-configuration
	var issuerURL string
	if app.IDPIssuer != "" {
		issuerURL = app.IDPIssuer
		log.Printf("[OIDCHandler] Using stored IDP Issuer URL: %s", issuerURL)
	} else {
		// Fallback to API hostname (shouldn't happen with auto-created apps)
		issuerURL = fmt.Sprintf("https://%s", app.APIHostname)
		log.Printf("[OIDCHandler] Using default IDP Issuer URL: %s", issuerURL)
	}

	// Initialize OIDC provider with issuer URL
	// The library will automatically fetch the discovery document from {issuerURL}/.well-known/openid-configuration
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	log.Printf("[OIDCHandler] OIDC provider initialized successfully")

	// Determine redirect URI
	redirectURI := app.RedirectURI
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("%s/app/%s/oidc/callback", baseURL, app.ID)
		log.Printf("[OIDCHandler] Using default redirect URI: %s", redirectURI)
	} else {
		log.Printf("[OIDCHandler] Using configured redirect URI: %s", redirectURI)
	}

	// Configure OAuth2
	oauth2Config := oauth2.Config{
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID}, // Only request openid scope
	}

	// Create ID token verifier
	verifier := provider.Verifier(&oidc.Config{
		ClientID: app.ClientID,
	})

	log.Printf("[OIDCHandler] OIDC handler initialized successfully")
	log.Printf("[OIDCHandler] Client ID: %s", app.ClientID)
	log.Printf("[OIDCHandler] Redirect URI: %s", redirectURI)

	return &OIDCHandler{
		App:          app,
		Session:      store,
		BaseURL:      baseURL,
		Provider:     provider,
		OAuth2Config: oauth2Config,
		Verifier:     verifier,
	}, nil
}

// Login displays the OIDC login page
func (h *OIDCHandler) Login(c fiber.Ctx) error {
	log.Printf("[OIDCHandler] Rendering login page for app: %s", h.App.Name)

	return c.Render("login", fiber.Map{
		"AppType":        "oidc",
		"AppID":          h.App.ID,
		"AppName":        h.App.Name,
		"RedirectURI":    h.OAuth2Config.RedirectURL,
		"APIHostname":    h.App.APIHostname,
		"AdminHostname":  getAdminHostname(h.App.APIHostname),
		"IntegrationKey": h.App.ClientID,
	})
}

// InitiateOIDC generates an OAuth2 authorization URL and redirects to Duo IDP
func (h *OIDCHandler) InitiateOIDC(c fiber.Ctx) error {
	log.Printf("[OIDCHandler] Initiating OIDC authentication for app: %s", h.App.Name)

	// Create session to store state and nonce
	sess, err := h.Session.Get(c)
	if err != nil {
		log.Printf("[OIDCHandler] Failed to get session: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Session error")
	}

	// Generate state parameter (for CSRF protection)
	state := generateRandomString(32)
	sess.Set("oidc_state", state)

	// Generate nonce (for replay attack protection)
	nonce := generateRandomString(32)
	sess.Set("oidc_nonce", nonce)

	if err := sess.Save(); err != nil {
		log.Printf("[OIDCHandler] Failed to save session: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Session error")
	}

	log.Printf("[OIDCHandler] Session ID: %s", sess.ID())
	log.Printf("[OIDCHandler] Generated state: %s", state)
	log.Printf("[OIDCHandler] Generated nonce: %s", nonce)

	// Build authorization URL with nonce
	authURL := h.OAuth2Config.AuthCodeURL(state, oidc.Nonce(nonce))

	log.Printf("[OIDCHandler] Redirecting to IDP: %s", authURL)
	return c.Redirect().To(authURL)
}

// Callback handles the OAuth2 callback from Duo IDP
func (h *OIDCHandler) Callback(c fiber.Ctx) error {
	log.Printf("[OIDCHandler] Received callback for app: %s", h.App.Name)

	// Get session
	sess, err := h.Session.Get(c)
	if err != nil {
		log.Printf("[OIDCHandler] Failed to get session: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Session error")
	}

	log.Printf("[OIDCHandler] Callback Session ID: %s", sess.ID())

	// Verify state parameter
	savedState := sess.Get("oidc_state")
	if savedState == nil {
		log.Printf("[OIDCHandler] No state found in session")
		return c.Status(fiber.StatusBadRequest).SendString("Invalid state: no state in session")
	}

	receivedState := c.Query("state")
	if receivedState != savedState.(string) {
		log.Printf("[OIDCHandler] State mismatch. Expected: %s, Got: %s", savedState, receivedState)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid state parameter")
	}

	log.Printf("[OIDCHandler] State verified successfully")

	// Check for error from IDP
	if errParam := c.Query("error"); errParam != "" {
		errDesc := c.Query("error_description")
		log.Printf("[OIDCHandler] Error from IDP: %s - %s", errParam, errDesc)
		return c.Status(fiber.StatusForbidden).SendString(fmt.Sprintf("Authentication error: %s - %s", errParam, errDesc))
	}

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		log.Printf("[OIDCHandler] No authorization code in callback")
		return c.Status(fiber.StatusBadRequest).SendString("Missing authorization code")
	}

	log.Printf("[OIDCHandler] Received authorization code: %s", code)

	ctx := context.Background()

	// Exchange authorization code for tokens
	oauth2Token, err := h.OAuth2Config.Exchange(ctx, code)
	if err != nil {
		log.Printf("[OIDCHandler] Failed to exchange code for token: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to exchange token: %v", err))
	}

	log.Printf("[OIDCHandler] Token exchange successful")

	// Extract ID token from OAuth2 token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Printf("[OIDCHandler] No id_token in token response")
		return c.Status(fiber.StatusInternalServerError).SendString("No id_token in response")
	}

	log.Printf("[OIDCHandler] Extracted ID token")

	// Verify ID token
	savedNonce := sess.Get("oidc_nonce")
	var nonceStr string
	if savedNonce != nil {
		nonceStr = savedNonce.(string)
	}

	idToken, err := h.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("[OIDCHandler] Failed to verify ID token: %v", err)
		return c.Status(fiber.StatusForbidden).SendString(fmt.Sprintf("Failed to verify ID token: %v", err))
	}

	// Verify nonce
	if idToken.Nonce != nonceStr {
		log.Printf("[OIDCHandler] Nonce mismatch. Expected: %s, Got: %s", nonceStr, idToken.Nonce)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid nonce")
	}

	log.Printf("[OIDCHandler] ID token verified successfully")

	// Extract claims from ID token
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		log.Printf("[OIDCHandler] Failed to extract claims: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to extract claims")
	}

	log.Printf("[OIDCHandler] Extracted claims: %+v", claims)

	// Get user info from userinfo endpoint (optional, for additional claims)
	userInfo, err := h.Provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		log.Printf("[OIDCHandler] Failed to get user info (non-fatal): %v", err)
	} else {
		var userInfoClaims map[string]interface{}
		if err := userInfo.Claims(&userInfoClaims); err == nil {
			// Merge userinfo claims with ID token claims
			for k, v := range userInfoClaims {
				if _, exists := claims[k]; !exists {
					claims[k] = v
				}
			}
			log.Printf("[OIDCHandler] Merged userinfo claims")
		}
	}

	// Extract standard claims
	userID := idToken.Subject
	userEmail := userID // Default to subject

	// Try to get the "user" claim first (Duo-specific, more human-readable)
	if user, ok := claims["user"].(string); ok && user != "" {
		userEmail = user
	} else if email, ok := claims["email"].(string); ok && email != "" {
		// Fall back to email claim if available
		userEmail = email
	}

	// Convert claims to JSON for session storage
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		log.Printf("[OIDCHandler] Failed to marshal claims: %v", err)
		claimsJSON = []byte("{}")
	}

	// Store authentication data in session
	sess.Set("authenticated", true)
	sess.Set("user_id", userID)
	sess.Set("user_email", userEmail)
	sess.Set("claims_json", string(claimsJSON))
	sess.Set("auth_time", time.Now().Unix())
	sess.Set("access_token", oauth2Token.AccessToken)
	sess.Set("token_type", oauth2Token.TokenType)

	// Clean up temporary session data
	sess.Delete("oidc_state")
	sess.Delete("oidc_nonce")

	if err := sess.Save(); err != nil {
		log.Printf("[OIDCHandler] Failed to save session: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Session error")
	}

	log.Printf("[OIDCHandler] User authenticated: %s", userEmail)

	// Redirect to success page
	return c.Redirect().To(fmt.Sprintf("/app/%s/oidc/success", h.App.ID))
}

// Success renders the success page after OIDC authentication
func (h *OIDCHandler) Success(c fiber.Ctx) error {
	log.Printf("[OIDCHandler] Rendering success page for app: %s", h.App.Name)

	// Get session
	sess, err := h.Session.Get(c)
	if err != nil {
		log.Printf("[OIDCHandler] Failed to get session: %v", err)
		return c.Redirect().To(fmt.Sprintf("/app/%s", h.App.ID))
	}

	// Check if authenticated
	authenticated := sess.Get("authenticated")
	if authenticated == nil || !authenticated.(bool) {
		log.Printf("[OIDCHandler] User not authenticated, redirecting to login")
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

	// Retrieve claims from JSON stored in session
	var claimsMap map[string]interface{}
	claimsJSONStr := sess.Get("claims_json")
	if claimsJSONStr != nil {
		if jsonStr, ok := claimsJSONStr.(string); ok {
			json.Unmarshal([]byte(jsonStr), &claimsMap)
		}
	}

	// Build a comprehensive response object
	responseData := map[string]interface{}{
		"sub":       userID,
		"user":      userEmail, // This will contain the username from the "user" claim
		"authTime":  authTimeStr,
		"claims":    claimsMap,
		"tokenType": sess.Get("token_type"),
	}

	// Format response data as JSON for display
	responseJSON, _ := json.MarshalIndent(responseData, "", "  ")

	return c.Render("success", fiber.Map{
		"AppType":        "oidc",
		"AppID":          h.App.ID,
		"AppName":        h.App.Name,
		"UserEmail":      userEmail,
		"AuthFactor":     "OpenID Connect",
		"AuthResult":     "success",
		"TokenData":      string(responseJSON),
		"ClaimsJSON":     string(responseJSON),
		"AdminHostname":  getAdminHostname(h.App.APIHostname),
		"IntegrationKey": h.App.ClientID,
	})
}

// Logout handles logout requests
func (h *OIDCHandler) Logout(c fiber.Ctx) error {
	log.Printf("[OIDCHandler] Handling logout request for app: %s", h.App.Name)

	// Get session
	sess, err := h.Session.Get(c)
	if err == nil {
		// Destroy session
		if err := sess.Destroy(); err != nil {
			log.Printf("[OIDCHandler] Failed to destroy session: %v", err)
		}
	}

	// Redirect to login page
	return c.Redirect().To(fmt.Sprintf("/app/%s", h.App.ID))
}

// generateRandomString generates a random base64 encoded string
func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
