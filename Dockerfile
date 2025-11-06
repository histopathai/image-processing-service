# Build stage
FROM golang:1.23-bullseye AS builder

WORKDIR /app

# Set GOTOOLCHAIN to auto to allow downloading Go 1.24
ENV GOTOOLCHAIN=auto

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o image-processing-service ./cmd/main.go

# Runtime stage with libvips
FROM debian:bullseye-slim

# Install dependencies (SADECE libvips-tools GEREKLİ)
# gcsfuse, fuse ve libvips-dev kaldırıldı
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libvips-tools \
    && rm -rf /var/lib/apt/lists/*

# Copy the binary from builder
COPY --from=builder /app/image-processing-service /app/image-processing-service

WORKDIR /app

# Sadece uygulamayı çalıştır
CMD ["/app/image-processing-service"]