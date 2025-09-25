# ===== Build stage =====
FROM golang:1.23 AS build
WORKDIR /src

# Modules cache
COPY go.mod go.sum ./
RUN go mod download

# Source
COPY . .

# --- bu yerda main paket yo'lini parametr qilamiz ---
ARG MAIN_PKG=./cmd/main.go   # agar sendeda boshqacha bo'lsa, compose'da override qilamiz
ARG VERSION=unknown
ARG COMMIT=dirty

# Build oldidan yo'l borligini tekshirib qo'yamiz (file yoki dir bo'lishi mumkin)
RUN [ -f "${MAIN_PKG}" ] || [ -d "${MAIN_PKG}" ] || (echo "MAIN_PKG not found: ${MAIN_PKG}"; ls -la; echo; find . -maxdepth 3 -type f -name 'main.go' -print; exit 1)

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
      -o /out/app "${MAIN_PKG}"

# ===== Runtime stage =====
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/app /app/app
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/app"]
