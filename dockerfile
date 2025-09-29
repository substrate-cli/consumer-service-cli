# consumer-service-cli/Dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary
RUN go build -o consumer-service-cli ./cmd/app

# Run stage
FROM alpine:latest
WORKDIR /app

# Install curl + docker-cli + nodejs + npm
RUN apk add --no-cache curl docker-cli nodejs npm docker-compose

# Check versions
RUN node --version && npm --version && npx --version && docker --version

# Copy the built binary
COPY --from=builder /app/consumer-service-cli .

# Command to run
CMD ["./consumer-service-cli"]
