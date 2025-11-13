# User Experience Toolkit - Go Edition

A modern Golang-based toolkit for Customer Success Engineers to test different Duo Policies using the v4 (Universal Prompt) WebSDK and Device Management Portal.

## Features

- **Multi-Application Support** - Configure and manage multiple Duo integrations
- **WebSDK v4 Universal Prompt** - Test Duo's modern authentication flow
- **Device Management Portal (DMP)** - Test device management with WebSDK v2 
- **Web-based Configuration Manager** - Add, edit, and manage applications through an intuitive UI
- **Dynamic Dashboard** - Automatically displays configured applications
- **Modern UI** - Built with vanilla HTML/CSS and Bootstrap 5

## Tech Stack

- **Backend**: Go 1.25+ with Fiber v3 framework
- **Frontend**: Vanilla HTML/CSS with Bootstrap 5
- **Authentication**: Duo Universal Go SDK

## Quick Start

### Prerequisites

- Go 1.25 or higher
- Duo account with Web SDK applications configured

### Installation

1. Clone or navigate to the repository:
```bash
cd user_experience_toolkit
```

2. Install dependencies:
```bash
go mod download
```

3. Build the application:
```bash
go build -o uet ./cmd/uet
```

4. Run the server:
```bash
./uet
```

5. Open your browser to [http://localhost:8080](http://localhost:8080)

6. Configure your first Duo application through the web interface at [http://localhost:8080/configure](http://localhost:8080/configure)

## Configuration

The toolkit uses a YAML-based configuration file (`config.yaml`) that supports multiple Duo applications. Configuration is best managed through the web interface at [http://localhost:8080/configure](http://localhost:8080/configure).

### Configuration File Format

```yaml
applications:
  - id: "unique-app-id"
    name: "Production V4"
    type: "v4"  # or "dmp"
    enabled: true
    client_id: "YOUR_CLIENT_ID"
    client_secret: "YOUR_CLIENT_SECRET"
    api_hostname: "api-xxxxxxxx.duosecurity.com"
    # V4-specific fields:
    redirect_uri: "http://localhost:8080/app/unique-app-id/callback"
    failmode: "closed"
    # DMP-specific field (only for type: dmp):
    akey: "GENERATE_YOUR_OWN_UNIQUE_SECRET_KEY"
```

### Application Types

**V4 (Universal Prompt)**
- Required fields: `client_id`, `client_secret`, `api_hostname`, `redirect_uri`, `failmode`
- The `redirect_uri` must match: `http://localhost:8080/app/{app-id}/callback`
- `failmode` can be "closed" (secure, default) or "open" (for testing)

**DMP (Device Management Portal)**
- Required fields: `client_id`, `client_secret`, `api_hostname`, `akey`
- The `akey` must be at least 40 characters long and randomly generated

### Web-Based Configuration Manager

1. Navigate to [http://localhost:8080/configure](http://localhost:8080/configure)
2. Click "Add Application" to create a new Duo integration
3. Fill in the application details:
   - **Name**: Descriptive name (e.g., "Production V4")
   - **Type**: Select "WebSDK V4" or "Device Management Portal"
   - **Enabled**: Toggle to enable/disable the application
   - **Credentials**: Enter your Duo application credentials from the Admin Panel
4. Click "Save Application"
5. The application will appear on the home dashboard if enabled

### Managing Applications

- **Edit**: Click the "Edit" button to modify an application's settings
- **Delete**: Click the "Delete" button to remove an application
- **Enable/Disable**: Toggle the enabled status to show/hide on the dashboard

## Usage

### Home Dashboard
The home page dynamically displays all enabled Duo applications. Each card shows:
- Application name
- Application type (V4 or DMP)
- API hostname

### Testing Applications

**Universal Prompt (V4)**
1. Click on a V4 application card from the home page
2. Enter any username and password
3. Complete the Duo 2FA prompt
4. View the authentication response token with user details

**Device Management Portal (DMP)**
1. Click on a DMP application card from the home page
2. Enter any username and password
3. Complete authentication in the Duo iframe
4. View success message

### Managing Applications
1. Click "Manage Applications" or the gear icon from the home page
2. View all configured applications in a table
3. Add, edit, or delete applications as needed

## Project Structure

```
.
├── cmd/uet/
│   └── main.go            # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go      # YAML configuration parser
│   ├── handlers/
│   │   ├── home.go        # Home page handler
│   │   ├── v4.go          # Universal Prompt handlers
│   │   ├── dmp.go         # Device Management Portal handlers
│   │   └── config.go      # Configuration API handlers
│   └── websdk2/
│       └── websdk2.go     # WebSDK v2 signature implementation
├── templates/
│   ├── layout.html        # Base layout template
│   ├── home.html          # Dynamic home dashboard
│   ├── v4_login.html      # V4 login page
│   ├── v4_success.html    # V4 success page
│   ├── dmp_login.html     # DMP login page
│   ├── dmp_iframe.html    # DMP iframe page
│   ├── dmp_success.html   # DMP success page
│   └── configure.html     # Configuration manager UI
├── static/
│   ├── css/
│   │   ├── style.css      # Custom styles with modal support
│   │   └── Duo-Frame.css  # Duo iframe styles
│   ├── js/
│   │   ├── Duo-Web-v2.js  # Duo WebSDK v2 JavaScript
│   │   └── theme.js       # Theme switcher
│   └── images/
│       └── logo.png       # Duo logo
├── config.yaml            # Multi-application configuration
├── go.mod                 # Go module dependencies
└── archive/               # Original PHP implementation
```

## Development

### Building
```bash
go build -o uet ./cmd/uet
```

### Running in Development
```bash
go run ./cmd/uet
```

### Running Tests
```bash
go test ./...
```

### Pre-commit Hooks

This project includes a Git pre-commit hook that automatically runs tests before each commit. This ensures code quality and prevents broken code from being committed.

The hook will:
- ✅ Run `go test ./...` before every commit
- ✅ Block the commit if any tests fail
- ✅ Show clear output about which tests passed/failed

**To bypass the hook** (use sparingly):
```bash
git commit --no-verify -m "Your message"
```

For more development guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md)

### Code Quality

Run static analysis tools:
```bash
# Format code
go fmt ./...

# Check for issues
go vet ./...

# Run staticcheck (install first: go install honnef.co/go/tools/cmd/staticcheck@latest)
staticcheck ./...

# Check cyclomatic complexity (install first: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest)
gocyclo -over 15 .
```

## What's New in v2.0

### Multi-Application Support
- Manage multiple Duo integrations from a single toolkit instance
- Dynamic dashboard displays all enabled applications
- Each application can be independently configured and toggled

### Modern Configuration Management
- YAML-based configuration for better structure
- Web-based CRUD interface with modal forms
- No need to manually edit configuration files
- Type-specific validation for V4 and DMP applications

### Improved Architecture
- Cleaner separation of concerns with internal packages
- Dynamic routing based on application ID
- Thread-safe configuration updates
- RESTful API for configuration management

## Migration from v1.x

If you're upgrading from the INI-based configuration:

1. The old `duo.conf` file is no longer used
2. Start the application - it will create a default `config.yaml`
3. Use the web interface at `/configure` to add your applications
4. Each application now has its own ID and can be managed independently

## Troubleshooting

### Configuration Not Loaded
- Ensure `config.yaml` exists in the application root directory
- Check file permissions (should be readable/writable by the application)
- Verify configuration format matches the YAML standard
- Use the web interface to add applications if the file is empty

### Authentication Failures
- Verify Duo credentials are correct
- Check that the redirect URI matches exactly in both the code and Duo Admin Panel
- Ensure your server's clock is synchronized (JWT validation is time-sensitive)
- Check the `failmode_v4` setting (set to "open" for testing if Duo is unavailable)

### Session Issues
- Sessions are stored in-memory by default
- Restarting the server will clear all sessions
- For production, consider using Redis or another persistent session store

## License

This toolkit is provided as-is for testing Duo Security integrations.

## Support

For issues related to Duo integration, please refer to:
- [Duo Web SDK Documentation](https://duo.com/docs/duoweb)
- [Duo Universal Prompt Guide](https://duo.com/docs/duoweb-v4)

