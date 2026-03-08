# 🧠 Image Processing Service

A Go-based microservice for processing medical whole slide images (WSIs). It supports tasks such as:

- Generating thumbnails
- Creating Deep Zoom Images (DZI)
- Extracting metadata (dimensions, format, file size)

Supported image formats include `.svs`, `.tif`, `.tiff`, `.jpg`, `.jpeg`, `.png`, `.ndpi`, `.scn`, `.bif`, `.vms`, `.vmu`, `.bmp`, and `.dng`.

---

## 🏗️ Architecture

- **Cloud Run Job**: Event-driven image processing
- **Pub/Sub Push**: Receives image processing requests via HTTP POST
- **GCS Mount**: Direct access to input/output buckets via FUSE mounts
- **CLI Mode**: Local usage via `himgproc` binary

---

## 📦 Dependencies

The following CLI tools must be installed on your system:

### ✅ macOS (via Homebrew)

```bash
brew install vips openslide exiftool
```

### ✅ Ubuntu/Debian

```bash
sudo apt update
sudo apt install -y libvips-tools libopenslide-bin libimage-exiftool-perl
```

> **Note:** Make sure `vips`, `openslide-show-properties`, and `exiftool` binaries are available in your `$PATH`.

### Automatic Installation

```bash
make deps
```

---

## 🖥️ CLI Usage (Local)

### Build & Install

```bash
# Install dependencies
make deps

# Build
make build

# Install (requires sudo)
sudo make install

# Now available globally
himgproc --help
```

### Command Line Options

| Option                | Short | Required | Default               | Description                                  |
| --------------------- | ----- | -------- | --------------------- | -------------------------------------------- |
| `--input`             | `-i`  | ✅       | —                     | Path to input image file                     |
| `--output`            | `-o`  | ❌       | `./output`            | Output directory for processed files         |
| `--image-id`          | —     | ❌       | derived from filename | Image ID                                     |
| `--version`           | —     | ❌       | `v2`                  | Processing version (`v1` or `v2`)            |
| `--log-level`         | —     | ❌       | `INFO`                | Log level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `--log-format`        | —     | ❌       | `text`                | Log format (`text` or `json`)                |
| `--tile-size`         | —     | ❌       | `256`                 | DZI Tile Size                                |
| `--overlap`           | —     | ❌       | `0`                   | DZI Overlap                                  |
| `--quality`           | —     | ❌       | `85`                  | DZI Quality level (1-100)                    |
| `--dzi-container`     | —     | ❌       | `zip`                 | DZI Container format (`zip` or `fs`)         |
| `--dzi-layout`        | —     | ❌       | `dz`                  | DZI Layout format                            |
| `--dzi-suffix`        | —     | ❌       | `jpg`                 | DZI Tile image suffix                        |
| `--dzi-compression`   | —     | ❌       | `0`                   | DZI Zip Compression Level (`0`-`9`)          |
| `--thumbnail-size`    | —     | ❌       | `256`                 | Thumbnail size (Width & Height)              |
| `--thumbnail-quality` | —     | ❌       | `90`                  | Thumbnail Quality level (1-100)              |

> **Configuration Priority:**
>
> 1. **CLI Flags** (Highest priority, overrides everything)
> 2. **Environment Variables / `.env` File** (Used if CLI flag is not provided)
> 3. **Default Values** (Configured as fallback when neither is provided)

### Examples

```bash
# Basic usage
himgproc -i ./slides/sample.svs -o ./results

# With explicit image ID
himgproc --input ./slides/biopsy.tiff --output ./processed --image-id biopsy-001

# With debug logging
himgproc -i ./image.png -o ./out --log-level DEBUG
```

### Output Structure

```
./output/{image-id}/
├── thumbnail.jpg       # Resized preview image
├── image.dzi           # Deep Zoom Image descriptor
├── image.zip           # DZI tiles archive (v2)
├── IndexMap.json       # Zip index map (v2)
└── result.json         # Processing result event JSON
```

The `result.json` content is also printed to **stdout**, making it pipe-friendly:

```bash
himgproc -i ./image.svs -o ./out 2>/dev/null | jq '.success'
```

### Makefile Commands

| Command               | Description                                             |
| --------------------- | ------------------------------------------------------- |
| `make deps`           | Install system dependencies (vips, openslide, exiftool) |
| `make deps-uninstall` | Uninstall system dependencies                           |
| `make build`          | Compile `himgproc` binary                               |
| `sudo make install`   | Install binary to `/usr/local/bin`                      |
| `make uninstall`      | Remove installed binary                                 |
| `make clean`          | Remove build artifacts                                  |

---

## 🚀 Cloud Deployment

### Architecture

```
Pub/Sub Topic → Cloud Run Job → GCS (output bucket) → Pub/Sub Result Topic
```

### Prerequisites

1. Google Cloud Project with required APIs enabled
2. GitHub repository secrets configured:
   - `WIF_PROVIDER`: Workload Identity Federation provider
   - `WIF_SERVICE_ACCOUNT`: Service account email
   - `TF_STATE_BUCKET`: Terraform state bucket
   - `ARTIFACT_REGISTRY_REPO_NAME`: Docker registry name
   - `GCP_REGION`: Cloud Run region (e.g., `us-central1`)

### Deployment

Pushing to `main` or `dev` branches triggers automatic deployment via GitHub Actions:

- `main` → **prod** environment
- `dev` → **dev** environment

For manual deployment:

```bash
# Go to GitHub Actions → Run "Deploy Image Processing Service" workflow
```

### Environment Variables (Cloud)

```env
APP_ENV=PROD
WORKER_TYPE=medium
PROJECT_ID=your-gcp-project-id
REGION=us-central1
ORIGINAL_BUCKET_NAME=histopath-original
PROCESSED_BUCKET_NAME=histopath-processed
IMAGE_PROCESS_RESULT_TOPIC_ID=image-processing-result
INPUT_MOUNT_PATH=/input
OUTPUT_MOUNT_PATH=/output
```

---

## 🔧 Legacy Local Mode (Env Vars)

Running `himgproc` without flags falls back to the legacy env var mode:

```bash
# Set up .env file
cp .env.example .env

# Run
go run cmd/main.go
```

Required env vars: `INPUT_IMAGE_ID`, `INPUT_ORIGIN_PATH`, `INPUT_PROCESSING_VERSION`, `INPUT_BUCKET_NAME`

---

## 🛠 Developer Notes

- Service scales to zero when idle to save costs
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

Service account needs permissions:

- `roles/pubsub.subscriber`
- `roles/pubsub.publisher`
- `roles/storage.objectViewer` (input bucket)
- `roles/storage.objectAdmin` (output bucket)

---

## 📬 Contact

For feedback, issues or contributions, feel free to open an issue or pull request.
