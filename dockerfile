# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.24.4-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o consumer-app ./cmd/app

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy with absolute paths
COPY --from=builder /app/consumer-app /root/consumer-app

# Set permissions
RUN chmod +x /root/consumer-app

# Copy .env file
COPY .env /root/.env

EXPOSE 8090

# Use absolute path in CMD
CMD ["/root/consumer-app"]
