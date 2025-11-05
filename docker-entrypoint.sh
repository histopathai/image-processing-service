#!/bin/bash
set -e

echo "Starting GCS Fuse mount process..."

# Check required environment variables
if [ -z "$GCS_ORIGINAL_BUCKET_NAME" ] || [ -z "$GCS_PROCESSED_BUCKET_NAME" ]; then
    echo "Error: GCS_ORIGINAL_BUCKET_NAME and GCS_PROCESSED_BUCKET_NAME must be set"
    exit 1
fi

# Mount input bucket (read-only)
echo "Mounting input bucket: $GCS_ORIGINAL_BUCKET_NAME to /mnt/input"
gcsfuse \
    --implicit-dirs \
    --file-mode=644 \
    --dir-mode=755 \
    --stat-cache-ttl=1h \
    --type-cache-ttl=1h \
    -o ro \
    "$GCS_ORIGINAL_BUCKET_NAME" /mnt/input

if [ $? -ne 0 ]; then
    echo "Failed to mount input bucket"
    exit 1
fi

echo "Input bucket mounted successfully"

# Mount output bucket (read-write)
echo "Mounting output bucket: $GCS_PROCESSED_BUCKET_NAME to /mnt/output"
gcsfuse \
    --implicit-dirs \
    --file-mode=644 \
    --dir-mode=755 \
    --stat-cache-ttl=10m \
    --type-cache-ttl=10m \
    "$GCS_PROCESSED_BUCKET_NAME" /mnt/output

if [ $? -ne 0 ]; then
    echo "Failed to mount output bucket"
    fusermount -u /mnt/input
    exit 1
fi

echo "Output bucket mounted successfully"

# Setup cleanup on exit
cleanup() {
    echo "Unmounting GCS buckets..."
    fusermount -u /mnt/output || true
    fusermount -u /mnt/input || true
    echo "Cleanup complete"
}

trap cleanup EXIT INT TERM

# Execute the main application
echo "Starting application..."
exec "$@"