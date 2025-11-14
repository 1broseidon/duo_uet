# Duo User Experience Toolkit

[![Go Version](https://img.shields.io/github/go-mod/go-version/1broseidon/duo_uet)](https://github.com/1broseidon/duo_uet)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](#docker)

> Self-hosted testing platform for Duo authentication flows with multi-tenant management and modern UI

A toolkit for Customer Success Engineers and technical teams to test and demonstrate Duo authentication policies across multiple integration types: Universal Prompt (WebSDK v4), Device Management Portal, SAML SSO, and OIDC SSO.

## Features

- **Multi-Tenant Management** - Configure multiple Duo tenants with isolated credentials
- **Multiple Auth Types** - WebSDK v4, DMP, SAML 2.0, OIDC in one platform
- **Web Configuration UI** - No manual config file editing required
- **Auto-Create Applications** - Generate Duo applications via Admin API
- **Optional Encryption** - AES-256-GCM encryption for config secrets at rest
- **Modern Design System** - Theme-aware UI with light/dark mode
- **Docker-First** - Production-ready containerization with multi-arch support

## Quick Start

### Using Docker (Recommended)

**Docker Compose:**
```bash
# Copy config
cp config.yaml.example config.yaml

# Edit config.yaml with your Duo credentials, then:
docker compose up -d
```

**Docker Run:**
```bash
docker run -d \
  --name duo-uet \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v $(pwd)/certs:/app/certs:ro \
  ghcr.io/1broseidon/duo_uet:latest
```

Access at `http://localhost:8080`

### From Source

**Prerequisites:**
- Go 1.25+
- Git

**Setup:**
```bash
# Clone and install
git clone https://github.com/1broseidon/duo_uet.git
cd duo_uet
go mod download

# Build and run
go build -o uet ./cmd/uet
./uet
```

Access at: `http://localhost:8080`

## Configuration

### Web UI (Recommended)

1. Navigate to `http://localhost:8080/configure`
2. Add a tenant with Admin API credentials
3. Create applications manually or auto-create from Duo Admin Panel
4. Applications appear on home dashboard when enabled

### Config File

The toolkit uses `config.yaml` for persistence. Structure:

```yaml
# Optional: Encrypt secrets at rest
encryption_enabled: false

# Multi-tenant support
tenants:
  - id: "tenant-1"
    name: "Production"
    api_hostname: "api-xxxxxxxx.duosecurity.com"
    admin_api_ikey: "DIXXXXXXXXXXXXXXXXXX"
    admin_api_secret: "your-secret-key"

# Applications auto-managed via UI or Admin API
applications:
  - id: "app-1"
    name: "WebSDK v4 Demo"
    type: "websdk"
    tenant_id: "tenant-1"
    enabled: true
    client_id: "DIXXXXXXXXXXXXXXXXXX"
    client_secret: "your-client-secret"
    # ... type-specific fields
```

See [config.yaml.example](config.yaml.example) for full schema.

### Encryption (Optional)

Protect secrets at rest with AES-256-GCM encryption:

```bash
# Enable in config.yaml
encryption_enabled: true

# Provide master key
export UET_MASTER_KEY="your-secure-password"

# Or use auto-generated key file
# (creates .uet_key with chmod 600)
./uet
```

See [config.yaml.example](config.yaml.example) for full configuration details.

## Supported Authentication Types

| Type | Description | Use Case |
|------|-------------|----------|
| **WebSDK v4** | Universal Prompt | Modern web applications |
| **DMP** | Device Management Portal | WebSDK v2 with device trust |
| **SAML 2.0** | Duo SSO SAML | Enterprise SSO integrations |
| **OIDC** | Duo SSO OpenID Connect | Modern SSO integrations |

Each type includes:
- Login flow simulation
- Token/claim inspection
- Success page with technical details
- Theme-aware UI

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Frontend                            │
│  Bulma CSS + Design System + Theme Switcher (Light/Dark)   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       Go Fiber v3 API                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Home     │  │ Config   │  │ Auth     │  │ Admin    │   │
│  │ Handler  │  │ Handler  │  │ Flows    │  │ API      │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Config Management                        │
│  YAML Storage + Optional AES-256-GCM Encryption             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Duo Integration                        │
│  Universal SDK + Admin API + SAML + OIDC Libraries          │
└─────────────────────────────────────────────────────────────┘
```

**Tech Stack:**
- **Backend:** Go 1.25, Fiber v3
- **Frontend:** Vanilla JS, Bulma CSS, Custom Design System
- **Storage:** YAML with optional encryption
- **Auth:** Duo Universal SDK, SAML 2.0, OIDC
- **Container:** Docker, Alpine Linux, multi-arch

## Development

### Local Development

```bash
# Run with auto-reload (requires air)
go install github.com/cosmtrek/air@latest
air

# Or standard go run
go run ./cmd/uet
```

### Testing

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Code Quality

```bash
# Format
go fmt ./...

# Vet
go vet ./...

# Static analysis (install: go install honnef.co/go/tools/cmd/staticcheck@latest)
staticcheck ./...

# Cyclomatic complexity (install: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest)
gocyclo -over 15 .
```

### Pre-commit Hooks

Automatically runs tests before each commit. To bypass:
```bash
git commit --no-verify -m "message"
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## Docker

### Pre-built Images

Pull and run from GitHub Container Registry:

```bash
docker pull ghcr.io/1broseidon/duo_uet:latest

docker run -d \
  --name duo-uet \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ghcr.io/1broseidon/duo_uet:latest
```

### Build Locally

```bash
docker build -t duo-uet:local .
docker run -d -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml:ro duo-uet:local
```

### Docker Compose

```yaml
version: '3.8'
services:
  uet:
    image: ghcr.io/1broseidon/duo_uet:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./certs:/app/certs:ro
    environment:
      UET_MASTER_KEY: ${UET_MASTER_KEY}  # Optional: for encrypted config
    restart: unless-stopped
```

### CI/CD

Automated builds trigger on version tags (`v*.*.*`). Tagged images available as:
- `v1.0.0` - Exact version
- `v1.0`, `v1` - Major/minor aliases
- `latest` - Latest release

## Project Structure

```
.
├── cmd/
│   ├── uet/              # Main application entry point
│   ├── encrypt-config/   # Config encryption utility
│   └── samltest/         # SAML testing utility
├── internal/
│   ├── config/           # YAML config + encryption
│   ├── crypto/           # AES-256-GCM encryption
│   ├── handlers/         # HTTP handlers (home, config, auth flows)
│   ├── duoadmin/         # Duo Admin API client
│   ├── websdk2/          # WebSDK v2 signature generation
│   └── saml/             # SAML request/response handling
├── templates/            # Go HTML templates
├── static/
│   ├── css/              # Design system, Bulma overrides
│   ├── js/               # Theme switcher, Duo WebSDK v2
│   └── images/           # Duo logo, assets
├── .github/workflows/    # CI/CD pipelines
├── CONTRIBUTING.md       # Development guidelines
└── config.yaml           # Runtime configuration
```

## Documentation

- **[Contributing](CONTRIBUTING.md)** - Development workflow and standards
- **[Config Examples](config.yaml.example)** - Full configuration schema

## Security

- **Config Encryption:** Optional AES-256-GCM for secrets at rest
- **Non-root Container:** Runs as UID 1000 in Docker
- **Secret Management:** Supports env vars and encrypted config
- **Pre-commit Tests:** Automated testing before commits
- **Dependency Updates:** Automated via Dependabot (if configured)

**Reporting vulnerabilities:** Open a GitHub issue or contact the maintainer.

## Common Issues

### Config not loading
```bash
# Check file exists and is readable
ls -la config.yaml

# Verify YAML syntax
cat config.yaml | python -c 'import yaml, sys; yaml.safe_load(sys.stdin)'
```

### Docker connectivity
```bash
# Verify port isn't in use
lsof -i :8080

# Check Docker logs
docker compose logs -f
```

### Authentication failures
- Verify Duo credentials in config
- Check redirect URI matches exactly
- Ensure server time is synchronized (JWT validation)
- Try `failmode: open` for testing

## Changelog

See [GitHub Releases](https://github.com/1broseidon/duo_uet/releases) for version history.

**Recent additions:**
- Docker containerization with multi-arch support
- GitHub Actions CI/CD for tagged releases
- Optional AES-256-GCM config encryption
- Redesigned success page for technical audiences
- Unified design system with light/dark themes

## License

MIT © [Your Name]

## Support

- 🐛 **Issues:** [GitHub Issues](https://github.com/1broseidon/duo_uet/issues)
- 📖 **Duo Docs:** [duo.com/docs](https://duo.com/docs)
- 💬 **Questions:** Open a discussion or issue

---

**Built for Customer Success Engineers** | **Powered by Duo Security** | **Go 1.25**
