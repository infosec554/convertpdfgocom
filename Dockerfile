# Golangning rasmidan foydalanish
FROM golang:1.18 AS builder

WORKDIR /app

# Modul fayllarni yuklash
COPY go.mod go.sum ./
RUN go mod download

# Manba kodni nusxalash
COPY . .

# Binary build qilish
RUN go build -o main .

# --- Run stage ---
FROM debian:bullseye

WORKDIR /app

# Builderdan binary olish
COPY --from=builder /app/main .

# Railway uchun dynamic port
ENV PORT=8080

# Portni expose qilish
EXPOSE ${PORT}

# Appni ishga tushirish
CMD ["./main"]
