FROM golang:1.23

WORKDIR /app

# Go mod fayllarni birinchi bo'lib copy qilish
COPY go.mod go.sum ./
RUN go mod download

# Keyin qolgan fayllarni copy qilish
COPY . .

RUN go build -o main ./cmd/main.go

CMD ["./main"]
