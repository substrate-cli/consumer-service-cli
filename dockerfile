# consumer-service-cli/Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o consumer-service-cli ./cmd/app

FROM alpine:latest
WORKDIR /app

# Install curl + docker-cli + nodejs + npm
RUN apk add --no-cache curl docker-cli nodejs npm docker-compose

# Check versions
RUN node --version && npm --version && npx --version && docker --version

# Copy the built binary
COPY --from=builder /app/consumer-service-cli .

CMD ["./consumer-service-cli"]
