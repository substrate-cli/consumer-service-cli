# consumer-service-cli/Dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary
RUN go build -o consumer-service-cli .

# Run stage
FROM alpine:latest
WORKDIR /app

# Copy the built binary
COPY --from=builder /app/consumer-service-cli .

# Command to run
CMD ["./consumer-service-cli"]
