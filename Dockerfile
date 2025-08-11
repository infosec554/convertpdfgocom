# Build stage
FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Run stage
FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/main .

# Railway dynamic port
ENV PORT=${PORT:-8080}
EXPOSE ${PORT}

CMD ["./main"]
