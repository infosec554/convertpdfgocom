# ===== Build stage =====
FROM golang:1.24 AS build
WORKDIR /src

# Modules cache
COPY go.mod go.sum ./
RUN go mod download

# Source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" -o /out/app ./cmd/main.go

# ===== Runtime stage =====
FROM debian:bookworm-slim
WORKDIR /app

# Install dependencies for PDF conversion (pdftoppm)
RUN apt-get update && apt-get install -y \
    poppler-utils \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy application artifacts
COPY --from=build /src/migrations /app/migrations
COPY --from=build /src/templates /app/templates
COPY --from=build /out/app /app/app

# Storage volume
VOLUME /app/storage

EXPOSE 8080

# Run as root (needed for apt/utils in this simplified setup, 
# or we can create a user, but for now strict permissions are handled by app)
ENTRYPOINT ["/app/app"]