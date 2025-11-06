# 🧠 Image Processing Service

This is a Go-based microservice for processing medical whole slide images (WSIs). It supports tasks such as:

- Generating thumbnails
- Creating Deep Zoom Images (DZI)
- Extracting metadata (dimensions, format, file size)

Supported image formats include `.svs`, `.tif`, `.tiff`, `.jpg`, `.jpeg`, `.png`, `.ndpi`, `.scn`, `.bif`, `.vms`, `.vmu`, and `.bmp`.

The service runs as a Cloud Run service and processes images triggered by Pub/Sub push subscriptions.

---

## 🏗️ Architecture

- **Cloud Run Service**: HTTP server listening on port 8080
- **Pub/Sub Push**: Receives image processing requests via HTTP POST
- **GCS Mount**: Direct access to input/output buckets via FUSE mounts
- **Event-Driven**: Publishes processing results back to Pub/Sub

---

## 📦 Dependencies

The following CLI tools must be installed on your system:

### ✅ Ubuntu

```bash
sudo apt update
sudo apt install -y \
  libvips-tools \
  libopenslide-bin \
  libimage-exiftool-perl
```

### ✅ macOS (via Homebrew)

```bash
brew install vips
brew install openslide
brew install exiftool
```

> Note: Make sure all binaries (`vips`, `openslide-show-properties`, `exiftool`) are available in your `$PATH`.

---

## 🚀 Running Locally

First, make sure your `.env` file is set properly. Example:

```env
APP_ENV=LOCAL

PROJECT_ID=your-gcp-project-id
GCS_ORIGINAL_BUCKET_NAME=your-original-bucket
GCS_PROCESSED_BUCKET_NAME=your-processed-bucket

IMAGE_PROCESSING_SUB_ID=your-subscription-id
IMAGE_PROCESS_RESULT_TOPIC_ID=your-result-topic-id

INPUT_MOUNT_PATH=./test-data/input
OUTPUT_MOUNT_PATH=./test-data/output

LOG_LEVEL=DEBUG
LOG_FORMAT=text

TILE_SIZE=256
OVERLAP=0
QUALITY=85
DZI_LAYOUT=dz
DZI_SUFFIX=jpeg

THUMBNAIL_SIZE=256
THUMBNAIL_QUALITY=90
```

Then build and run:

```bash
go mod tidy
go run cmd/main.go
```

The server will start on port 8080 (or the port specified in `PORT` env var).

---

## 🔍 Endpoints

### Health Check
```bash
GET http://localhost:8080/health

Response:
{
  "status": "healthy",
  "time": "2025-01-15T10:30:00Z"
}
```

### Pub/Sub Push Endpoint
```bash
POST http://localhost:8080/

Body: Pub/Sub push message format
{
  "message": {
    "data": "base64-encoded-event-data",
    "attributes": {
      "event_type": "image.processing.requested.v1",
      "image_id": "abc-123"
    },
    "messageId": "123456"
  },
  "subscription": "projects/PROJECT_ID/subscriptions/SUB_ID"
}
```

---

## 🔧 Testing with cURL

You can simulate a Pub/Sub push message locally:

```bash
# First, base64 encode your event data
EVENT_DATA='{"EventID":"test-123","EventType":"image.processing.requested.v1","Timestamp":"2025-01-15T10:00:00Z","image-id":"test-image-1","origin-path":"test-folder/image.svs"}'
ENCODED_DATA=$(echo -n "$EVENT_DATA" | base64)

# Send the request
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d "{
    \"message\": {
      \"data\": \"$ENCODED_DATA\",
      \"attributes\": {
        \"event_type\": \"image.processing.requested.v1\",
        \"image_id\": \"test-image-1\"
      },
      \"messageId\": \"test-msg-123\"
    },
    \"subscription\": \"projects/test/subscriptions/test-sub\"
  }"
```

---

## 🚢 Deployment

### Prerequisites
1. Google Cloud Project with required APIs enabled
2. GitHub repository secrets configured:
   - `GCP_SA_KEY`: Service account JSON key
   - `GCP_PROJECT_ID`: Your GCP project ID
   - `GCP_REGION`: Cloud Run region (e.g., `us-central1`)
   - `GCS_ORIGINAL_BUCKET_NAME`: Input bucket name
   - `GCS_PROCESSED_BUCKET_NAME`: Output bucket name
   - `GCP_IMAGE_PROCESSING_SUB_ID`: Pub/Sub subscription ID
   - `GCP_IMAGE_PROCESS_RESULT_TOPIC_ID`: Result topic ID

### Deployment Steps

1. **Build and Push Docker Image**
   - Go to GitHub Actions
   - Run "Build Image Processing Service" workflow
   - This builds and pushes the image to Artifact Registry

2. **Deploy to Cloud Run**
   - Run "Deploy Image Processing Service" workflow
   - This deploys the service and configures Pub/Sub push subscription

### Manual Deployment

```bash
# Build and push image
docker build -t us-central1-docker.pkg.dev/PROJECT_ID/histopath-docker-repo/image-processing-service:latest .
docker push us-central1-docker.pkg.dev/PROJECT_ID/histopath-docker-repo/image-processing-service:latest

# Deploy to Cloud Run
gcloud run deploy image-processing-service \
  --image=us-central1-docker.pkg.dev/PROJECT_ID/histopath-docker-repo/image-processing-service:latest \
  --region=us-central1 \
  --platform=managed \
  --allow-unauthenticated \
  --port=8080 \
  --cpu=2 \
  --memory=4Gi \
  --timeout=3600 \
  --concurrency=10 \
  --add-volume=name=input-bucket,type=cloud-storage,bucket=INPUT_BUCKET \
  --add-volume-mount=volume=input-bucket,mount-path=/gcs/INPUT_BUCKET \
  --set-env-vars=APP_ENV=PROD,...

# Configure Pub/Sub push subscription
SERVICE_URL=$(gcloud run services describe image-processing-service --region=us-central1 --format='value(status.url)')

gcloud pubsub subscriptions create image-processing-sub \
  --topic=image-processing-requests \
  --push-endpoint="${SERVICE_URL}/" \
  --ack-deadline=600
```

---

## 🛠 Developer Notes

- The service scales to zero when idle to save costs
- Processing timeout is set to 3600 seconds (1 hour)
- Concurrent processing limit is 10 instances
- Health checks ensure the service is responsive
- Pub/Sub push automatically retries failed messages

---

## 📊 Monitoring

Monitor your service in the Google Cloud Console:
- **Cloud Run Metrics**: Request count, latency, error rate
- **Logs**: View processing logs in Cloud Logging
- **Pub/Sub Metrics**: Message delivery, ack/nack rates
- **Cloud Storage**: Monitor bucket usage and operations

---

## 🔒 Security

- Service account needs permissions:
  - `roles/pubsub.subscriber`
  - `roles/pubsub.publisher`
  - `roles/storage.objectViewer` (input bucket)
  - `roles/storage.objectAdmin` (output bucket)

---

## 📬 Contact

For feedback, issues or contributions, feel free to open an issue or pull request.