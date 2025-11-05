# Build stage
FROM golang:1.23-bullseye AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o image-processing-service .

# Runtime stage with GCS Fuse and libvips
FROM debian:bullseye-slim

# Install dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    fuse \
    libvips-tools \
    libvips-dev \
    && rm -rf /var/lib/apt/lists/*

# Install gcsfuse
RUN export GCSFUSE_REPO=gcsfuse-`lsb_release -c -s` && \
    echo "deb https://packages.cloud.google.com/apt $GCSFUSE_REPO main" | tee /etc/apt/sources.list.d/gcsfuse.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
    apt-get update && \
    apt-get install -y gcsfuse && \
    rm -rf /var/lib/apt/lists/*

# Create mount directories
RUN mkdir -p /mnt/input /mnt/output

# Copy the binary from builder
COPY --from=builder /app/image-processing-service /app/image-processing-service

# Copy entrypoint script
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

WORKDIR /app

# Use entrypoint for GCS mounting
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/app/image-processing-service"]