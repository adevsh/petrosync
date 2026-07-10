# Multi-stage Docker build for Go application using UBI Micro for minimal size
# UBI (Universal Base Image) Micro is RHEL-based and extremely small (~40MB)

# Stage 1: Build the Go application
FROM golang:1.25 AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
# CGO_ENABLED=0 for static binary that works with UBI Micro
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -trimpath \
    -o /app/petrosync \
    ./cmd

# Stage 2: Create minimal runtime image using UBI Micro
# UBI Micro is ~40MB and based on RHEL/Fedora family
FROM registry.access.redhat.com/ubi9/ubi-micro:latest

# Set working directory
WORKDIR /app

# Copy CA certificates from builder for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/pki/tls/certs/ca-bundle.crt

# Copy binary from builder stage
COPY --from=builder /app/petrosync /app/petrosync

# Create necessary directories with proper permissions
# UBI Micro runs as non-root by default (UID 1001)
RUN mkdir -p /app/logs /app/uploads && \
    chown -R 1001:0 /app && \
    chmod -R g=u /app

# Switch to non-root user (UBI Micro default user)
USER 1001

# Expose port
EXPOSE 8080

# Note: UBI Micro doesn't include curl, so healthcheck is handled by orchestrator
# For Docker Compose, use external healthcheck or add curl in a larger UBI image

# Command to run the application
CMD ["/app/petrosync"]
