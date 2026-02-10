#!/bin/bash
# Local test script for image processing service
# Tests the refactored storage abstraction layer

set -e

echo "🧪 Testing Image Processing Service Locally"
echo "============================================"

# Setup
TEST_IMAGE="test.png"
IMAGE_ID="test-$(date +%s)"
OUTPUT_DIR="./test-data/output"

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo ""
echo "📋 Test Configuration:"
echo "  Input: ./test-data/input/$TEST_IMAGE"
echo "  Image ID: $IMAGE_ID"
echo "  Output: $OUTPUT_DIR/$IMAGE_ID/"
echo "  Processing Version: v2 (zip container)"
echo ""

# Set environment variables
export APP_ENV=LOCAL
export LOG_LEVEL=DEBUG
export LOG_FORMAT=text

# Storage configuration
export INPUT_MOUNT_PATH=./test-data/input
export OUTPUT_MOUNT_PATH=./test-data/output

# Job input variables
export INPUT_IMAGE_ID="$IMAGE_ID"
export INPUT_ORIGIN_PATH="$TEST_IMAGE"  # Relative path in input storage
export INPUT_PROCESSING_VERSION="v2"
export INPUT_BUCKET_NAME="test-bucket"  # Not used in local mode

# Processing configuration
export DZI_CONTAINER=zip
export DZI_COMPRESSION=0
export TILE_SIZE=256
export OVERLAP=0
export QUALITY=85
export DZI_LAYOUT=dz
export DZI_SUFFIX=jpg
export THUMBNAIL_SIZE=256
export THUMBNAIL_QUALITY=90

echo "🚀 Running image processing job..."
echo ""

# Run the job
if /tmp/image-processing-job; then
    echo ""
    echo "✅ Job completed successfully!"
    echo ""
    echo "📁 Output files:"
    ls -lh "$OUTPUT_DIR/$IMAGE_ID/" 2>/dev/null || echo "  (no output directory created)"
    echo ""
    
    # Validate expected outputs for v2
    EXPECTED_FILES=("thumbnail.jpg" "image.zip" "image.dzi" "IndexMap.json")
    echo "🔍 Validating outputs..."
    ALL_GOOD=true
    for file in "${EXPECTED_FILES[@]}"; do
        if [ -f "$OUTPUT_DIR/$IMAGE_ID/$file" ]; then
            SIZE=$(ls -lh "$OUTPUT_DIR/$IMAGE_ID/$file" | awk '{print $5}')
            echo "  ✓ $file ($SIZE)"
        else
            echo "  ✗ $file (missing)"
            ALL_GOOD=false
        fi
    done
    
    if [ "$ALL_GOOD" = true ]; then
        echo ""
        echo "🎉 All expected outputs present!"
    else
        echo ""
        echo "⚠️  Some outputs are missing"
        exit 1
    fi
else
    echo ""
    echo "❌ Job failed!"
    exit 1
fi
