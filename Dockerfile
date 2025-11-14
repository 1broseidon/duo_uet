# Multi-stage build for Duo User Experience Toolkit
# Stage 1: Build the Go binary
FROM golang:1.25.0-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
# CGO_ENABLED=0 for static binary, useful for scratch/distroless images
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o uet \
    ./cmd/uet

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS requests and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for running the application
RUN addgroup -g 1000 uet && \
    adduser -D -u 1000 -G uet uet

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/uet /app/uet

# Copy static assets and templates
COPY --chown=uet:uet static /app/static
COPY --chown=uet:uet templates /app/templates

# Copy example config (users will mount their own config.yaml)
COPY --chown=uet:uet config.yaml.example /app/config.yaml.example

# Create directory for certs (optional, can be volume mounted)
RUN mkdir -p /app/certs && chown uet:uet /app/certs

# Create directory for config (will be volume mounted in production)
RUN mkdir -p /app/config && chown uet:uet /app/config

# Switch to non-root user
USER uet

# Expose application port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Run the application
CMD ["/app/uet"]
