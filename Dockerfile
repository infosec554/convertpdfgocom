FROM golang:1.23

WORKDIR /app

COPY go.mod go.sum ./

ENV GOPROXY=https://proxy.golang.org,direct

RUN go mod download

COPY . .

RUN go build -o main ./cmd/main.go

CMD ["./main"]
