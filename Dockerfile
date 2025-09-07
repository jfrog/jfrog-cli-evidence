# Multi-stage Dockerfile for JFrog CLI Evidence

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev') \
    -X main.Commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') \
    -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
    -o jfrog-evidence \
    evidence/cmd/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates git

# Create non-root user
RUN addgroup -g 1000 -S jfrog && \
    adduser -u 1000 -S jfrog -G jfrog

# Set working directory
WORKDIR /home/jfrog

# Copy binary from builder
COPY --from=builder /app/jfrog-evidence /usr/local/bin/jfrog-evidence

# Change ownership
RUN chown -R jfrog:jfrog /home/jfrog

# Switch to non-root user
USER jfrog

# Set entrypoint
ENTRYPOINT ["jfrog-evidence"]

# Default command (show help)
CMD ["--help"]
