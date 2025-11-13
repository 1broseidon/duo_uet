package handlers

import (
	"user_experience_toolkit/internal/config"

	"github.com/gofiber/fiber/v3"
)

type HomeHandler struct {
	Config *config.Config
}

func NewHomeHandler(cfg *config.Config) *HomeHandler {
	return &HomeHandler{Config: cfg}
}

func (h *HomeHandler) Index(c fiber.Ctx) error {
	enabledApps := h.Config.GetEnabledApplications()
	tenants := h.Config.GetAllTenants()

	// Create a map of tenant ID to tenant name
	tenantMap := make(map[string]string)
	for _, tenant := range tenants {
		tenantMap[tenant.ID] = tenant.Name
	}

	return c.Render("home", fiber.Map{
		"Applications": enabledApps,
		"Tenants":      tenants,
		"HasTenants":   len(tenants) > 0,
		"TenantMap":    tenantMap,
	})
}
