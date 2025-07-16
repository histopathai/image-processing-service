# 🧠 Medical Image Annotator - Image Processing Service

This is a Go-based microservice for processing medical whole slide images (WSIs). It supports tasks such as:

- Generating thumbnails
- Creating Deep Zoom Images (DZI)
- Extracting metadata (dimensions, format, file size)

Supported image formats include `.svs`, `.tif`, `.tiff`, `.jpg`, `.jpeg`, and `.png`.  
The service integrates with Google Cloud Storage and Firestore.

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

## 🚀 Running the Service

First, make sure your `.env` file is set properly. Example:

```env
ENV=LOCAL
GIN_MODE=release

SERVER_PORT=4141
SUPPORTED_FORMATS=svs,tif,tiff,jpg,jpeg,png

GCP_PROJECT_ID=your-gcp-project-id
GCP_LOCATION=us-central1
GCP_BUCKET=your-bucket
GCP_FIRESTORE_COLLECTION=images
GOOGLE_APPLICATION_CREDENTIALS=./.credentials/gac.json

TILE_SIZE=256
OVERLAP=0
LAYOUT=dz
QUALITY=75
SUFFIX=.jpg
THUMBNAIL_SIZE=256
```

Then build and run:

```bash
go mod tidy
go run cmd/main.go
```

The server will start on the specified port (e.g., `http://localhost:4141`).

---

## 📡 Example Request

You can test the upload endpoint using `curl`:

```bash
curl -X POST http://localhost:4141/upload \
  -H "Content-Type: application/json" \
  -d '{
    "image_path": "/absolute/path/to/MSB-00030-01-02.svs",
    "dataset_info": {
      "file_name": "MSB-00030-01-02.svs",
      "file_uid": "MSB-00030-01-02.svs",
      "dataset_name": "CMB-BRCA",
      "organ_type": "breast",
      "disease_type": "carcinoma",
      "classification": "carcinoma",
      "sub_type": "ductal",
      "grade": ""
    }
  }'
```

The service will:

- Extract image metadata via `exiftool`
- Generate a thumbnail
- Generate DZI tiles in the configured layout
- Upload outputs to GCS and save metadata to Firestore

---

## 🛠 Developer Notes

- The system can be extended to support **job tracking** using `job_id` values to monitor long-running operations.
- Firestore can be used as a backend to store job states, logs, and results.
- For heavy processing, consider integrating a background worker or queue system.

---

## 📬 Contact

For feedback, issues or contributions, feel free to open an issue or pull request.