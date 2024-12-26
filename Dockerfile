FROM golang:1.23.4-alpine3.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o lru-cache-service ./cmd/app/main.go

FROM alpine:3.21

WORKDIR /root/

COPY --from=builder /app/lru-cache-service .

# При запуске контейнера необходимо использовать 0.0.0.0, а не localhost  
ENV SERVER_HOST_PORT=0.0.0.0:8080

CMD ["./lru-cache-service"]
