# Docker Deployment Guide

This guide covers building and deploying the Duo User Experience Toolkit using Docker.

## Quick Start

### Using Docker Compose (Recommended for Local Development)

1. **Copy and configure your config file:**
   ```bash
   cp config.yaml.example config.yaml
   # Edit config.yaml with your Duo credentials and applications
   ```

2. **Start the application:**
   ```bash
   docker-compose up -d
   ```

3. **View logs:**
   ```bash
   docker-compose logs -f
   ```

4. **Stop the application:**
   ```bash
   docker-compose down
   ```

### Using Docker CLI

1. **Build the image:**
   ```bash
   docker build -t uet:latest .
   ```

2. **Run the container:**
   ```bash
   docker run -d \
     --name uet \
     -p 8080:8080 \
     -v $(pwd)/config.yaml:/app/config.yaml:ro \
     -v $(pwd)/certs:/app/certs:ro \
     uet:latest
   ```

3. **View logs:**
   ```bash
   docker logs -f uet
   ```

4. **Stop the container:**
   ```bash
   docker stop uet
   docker rm uet
   ```

## Using Pre-built Images from GitHub Container Registry

Once a tagged version is released, you can pull and run pre-built images:

```bash
# Pull the latest version
docker pull ghcr.io/OWNER/user_experience_toolkit:latest

# Pull a specific version
docker pull ghcr.io/OWNER/user_experience_toolkit:v1.0.0

# Run the pre-built image
docker run -d \
  --name uet \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ghcr.io/OWNER/user_experience_toolkit:latest
```

## Configuration

### Required Files

- **config.yaml**: Your application configuration (required)
  - Mount to: `/app/config.yaml`
  - Can be plaintext or encrypted

### Optional Files

- **certs/**: TLS certificates for HTTPS (optional)
  - Mount to: `/app/certs`

- **.uet_key**: Encryption key file (if using encrypted config)
  - Mount to: `/app/.uet_key`
  - **Important**: Never commit this file to version control

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `UET_MASTER_KEY` | Master key for encrypted config | - |
| `CONFIG_FILE` | Path to config file | `/app/config.yaml` |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |
| `TZ` | Timezone | `UTC` |

### Using Encrypted Configuration

If you're using encrypted configuration:

```yaml
# docker-compose.yml
services:
  uet:
    environment:
      UET_MASTER_KEY: "your-secure-master-key"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      # OR mount the key file:
      # - ./.uet_key:/app/.uet_key:ro
```

## Production Deployment

### Docker Compose (Production)

```yaml
version: '3.8'

services:
  uet:
    image: ghcr.io/OWNER/user_experience_toolkit:v1.0.0
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./certs:/app/certs:ro
    environment:
      UET_MASTER_KEY: ${UET_MASTER_KEY}
      TZ: America/New_York
    restart: always
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/"]
      interval: 30s
      timeout: 3s
      start_period: 5s
      retries: 3
```

Use an `.env` file for sensitive environment variables:

```bash
# .env
UET_MASTER_KEY=your-secure-master-key
```

### Kubernetes Deployment

Example Kubernetes manifests:

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: uet
spec:
  replicas: 2
  selector:
    matchLabels:
      app: uet
  template:
    metadata:
      labels:
        app: uet
    spec:
      containers:
      - name: uet
        image: ghcr.io/OWNER/user_experience_toolkit:v1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: UET_MASTER_KEY
          valueFrom:
            secretKeyRef:
              name: uet-secrets
              key: master-key
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
          readOnly: true
      volumes:
      - name: config
        configMap:
          name: uet-config
---
apiVersion: v1
kind: Service
metadata:
  name: uet
spec:
  selector:
    app: uet
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

## GitHub Actions CI/CD

The repository includes a GitHub Action that automatically builds and pushes Docker images when you create a new tag:

1. **Create and push a tag:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub Actions will automatically:**
   - Build the Docker image for multiple architectures (amd64, arm64)
   - Tag it with the version number
   - Push it to GitHub Container Registry
   - Create artifact attestations

3. **Available tags:**
   - `v1.0.0` - Specific version
   - `v1.0` - Minor version
   - `v1` - Major version
   - `latest` - Latest release

## Image Details

- **Base Image**: Alpine Linux (minimal, secure)
- **User**: Non-root user `uet` (UID 1000, GID 1000)
- **Exposed Port**: 8080
- **Health Check**: HTTP GET on `/`
- **Supported Architectures**: linux/amd64, linux/arm64

## Troubleshooting

### Container won't start

```bash
# Check logs
docker logs uet

# Check if config file is mounted correctly
docker exec uet ls -la /app/config.yaml
```

### Permission errors

Ensure the mounted files are readable by UID 1000:

```bash
chmod 644 config.yaml
```

### Health check failures

Test the health endpoint:

```bash
docker exec uet wget --no-verbose --tries=1 --spider http://localhost:8080/
```

## Security Best Practices

1. **Never commit secrets**: Use encrypted config or environment variables
2. **Use read-only mounts**: Mount config files as read-only (`:ro`)
3. **Keep images updated**: Regularly pull new versions
4. **Use specific tags**: Avoid using `latest` in production
5. **Scan images**: Use Docker Scout or similar tools to scan for vulnerabilities
6. **Limit resources**: Set memory/CPU limits in production

## Building from Source

If you need to customize the image:

```bash
# Build with custom arguments
docker build \
  --build-arg VERSION=custom \
  --build-arg COMMIT_SHA=$(git rev-parse HEAD) \
  -t uet:custom \
  .

# Build for specific platform
docker build --platform linux/arm64 -t uet:arm64 .
```

## Support

For issues or questions:
- Check the main [README.md](README.md)
- Review logs: `docker logs uet`
- Open an issue on GitHub
