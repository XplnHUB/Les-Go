# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server_bin ./server/main.go

# Final stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server_bin .
EXPOSE 8080
CMD ["./server_bin"]
