# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the server binary
# CGO_ENABLED=0 ensures a static binary for Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server_bin ./server/main.go

# Production stage
FROM alpine:latest

# Security: Add a non-root user
RUN adduser -D -u 1000 lesgo
USER lesgo

WORKDIR /home/lesgo/

# Copy the binary from the builder stage
COPY --from=builder /app/server_bin .

# Documentation
EXPOSE 80

# Run the binary
ENTRYPOINT ["./server_bin"]
