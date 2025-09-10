
FROM golang:1.23 AS build
WORKDIR /src

# 1) Modules first for cache (tezroq build uchun)
COPY go.mod go.sum ./
RUN go mod download

# 2) Copy source code
COPY . .

ARG VERSION=unknown
ARG COMMIT=dirty
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
      -o /out/app ./cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/app /app/app

# 2) Port
EXPOSE 8080

# 3) Drop privileges (nonroot is already default in distroless)
USER nonroot:nonroot

# 4) Entry
ENTRYPOINT ["/app/app"]
