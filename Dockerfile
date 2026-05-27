# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /lemon-chat ./cmd/lemon-chat

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /lemon-chat /usr/local/bin/lemon-chat
COPY static/ ./static/

VOLUME ["/data"]

EXPOSE 8080

ENTRYPOINT ["lemon-chat", "--config", "/data/lemon.toml"]
