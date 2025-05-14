FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -tags "go_tarantool_ssl_disable" -o /bin/bot ./cmd/bot/main.go 
FROM alpine:3.17
WORKDIR /app
COPY --from=builder /bin/bot /app/bot
ENTRYPOINT ["/app/bot"]