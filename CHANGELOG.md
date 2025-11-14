# Changelog


All notable changes to this project will be documented in this file.
## [v1.0.3] - 2025-11-14

## [v1.0.3] - 2025-11-14

### Added
- **Encryption tool** - Encrypt sensitive fields in YAML configuration files using the `encrypt-config` command
- **New application handlers** - Added handlers for DMP, SAML, OIDC, and Websdk applications in `cmd/uet/main.go`
- **Dynamic application routes** - Routes are now dynamically generated based on application type and ID

### Changed
- **Configuration management** - Improved configuration loading, saving, and validation in `internal/config/config.go`
- **Error handling** - Enhanced error messages and handling for better user experience
- **Docker containerization** - Updated Dockerfile for better multi-arch support and build optimizations

### Fixed
- **Ignore patterns** - Fixed ignore patterns to prevent exclusion of cmd/ source directories
- **CHANGELOG formatting** - Consolidated CHANGELOG into v1.0.0 release and updated formatting for future releases

### Security
- **Encryption** - Added encryption for sensitive fields in configuration files using a master key or .uet_key file


The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-11-14

### Added
- Multi-tenant Duo configuration management
- Support for WebSDK v4 (Universal Prompt)
- Support for Device Management Portal (DMP)
- Support for SAML 2.0 SSO
- Support for OIDC SSO
- Web-based configuration UI
- Auto-create applications via Duo Admin API
- Optional AES-256-GCM config encryption
- Docker containerization with multi-arch support (amd64, arm64)
- GitHub Actions CI/CD pipeline
- Automated changelog generation with AI (Cerebras)
- Pre-commit hooks for automated testing
- Modern design system with light/dark theme
- Technical details card for authentication results

### Changed
- Redesigned success page for engineering audiences
- Simplified navigation (removed Home tab)
- Login page minimization
- Icon-only action buttons with hover glow effects

### Security
- AES-256-GCM encryption for config secrets at rest
- Non-root Docker container (UID 1000)
- Environment variable support for sensitive data
