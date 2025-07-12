# Multi-stage build for gones NES emulator
FROM golang:1.23-alpine AS builder

# Set up build environment
WORKDIR /app

# Install build dependencies including SDL2
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    pkgconf \
    sdl2-dev

# Copy go modules first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with version information
ARG VERSION=docker
ARG GIT_COMMIT=unknown
ARG BUILD_TIME
ARG BUILD_USER=docker

RUN CGO_ENABLED=1 go build \
    -ldflags "-X gones/internal/version.Version=${VERSION} \
             -X gones/internal/version.GitCommit=${GIT_COMMIT} \
             -X gones/internal/version.BuildTime=${BUILD_TIME} \
             -X gones/internal/version.BuildUser=${BUILD_USER}" \
    -o gones ./cmd/gones

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    sdl2 \
    sdl2-dev \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S gones && \
    adduser -u 1001 -S gones -G gones

# Create directories for emulator data
RUN mkdir -p /app/roms /app/saves /app/states /app/screenshots /app/config && \
    chown -R gones:gones /app

# Copy binary from builder stage
COPY --from=builder /app/gones /usr/local/bin/gones
RUN chmod +x /usr/local/bin/gones

# Copy sample configuration if it exists
COPY --from=builder --chown=gones:gones /app/config/ /app/config/ 2>/dev/null || true

# Switch to non-root user
USER gones
WORKDIR /app

# Expose any ports if needed (for web interface, debugging, etc.)
# EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD gones -version || exit 1

# Set default command
ENTRYPOINT ["gones"]
CMD ["--help"]

# Metadata
LABEL \
    org.opencontainers.image.title="gones" \
    org.opencontainers.image.description="Go NES Emulator - A cycle-accurate Nintendo Entertainment System emulator written in Go" \
    org.opencontainers.image.version="${VERSION}" \
    org.opencontainers.image.revision="${GIT_COMMIT}" \
    org.opencontainers.image.created="${BUILD_TIME}" \
    org.opencontainers.image.source="https://github.com/your-org/gones" \
    org.opencontainers.image.url="https://github.com/your-org/gones" \
    org.opencontainers.image.documentation="https://github.com/your-org/gones/blob/main/README.md" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.vendor="gones contributors"