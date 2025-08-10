# Golangning rasmidan foydalanish
FROM golang:1.18

# Ishchi katalogni o'rnatish
WORKDIR /app

# Go modlarni o'rnatish
COPY go.mod go.sum ./
RUN go mod tidy

# Kodekni konteynerga nusxalash
COPY . .

# Kodekni kompilyatsiya qilish
RUN go build -o main .

# Portni eslatib o'tish
EXPOSE 8080

# Ilovani ishga tushurish
CMD ["./main"]
