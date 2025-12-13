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
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

# Migrations papkasini ko'chirish
COPY --from=build /src/migrations /app/migrations
COPY --from=build /out/app /app/app

# Storage papkasi uchun
VOLUME /app/storage

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/app"]