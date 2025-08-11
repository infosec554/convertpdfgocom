# --- Build stage ---
FROM golang:1.20 AS builder

WORKDIR /app

# Go mod fayllarni avval nusxalash (caching uchun)
COPY go.mod go.sum ./
RUN go mod download

# Manba kodni nusxalash
COPY . .

# Binary build qilish
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# --- Run stage ---
FROM alpine:3.18

WORKDIR /app

# Timezone va SSL sertifikatlar uchun (Postgres/HTTPS ishlashi uchun)
RUN apk --no-cache add ca-certificates tzdata

# Builderdan binary olish
COPY --from=builder /app/main .

# Railway uchun dynamic port
ENV PORT=8080

# Portni expose qilish
EXPOSE ${PORT}

# Appni ishga tushirish
CMD ["./main"]
