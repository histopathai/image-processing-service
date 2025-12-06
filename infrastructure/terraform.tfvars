# Environment (prod or dev)
environment = "prod"

# Docker image tag to deploy
image_tag = "latest"

# Logging
log_level = "INFO"

# DZI Configuration
tile_size   = 512
overlap     = 0
quality     = 85
dzi_layout  = "dz"
dzi_suffix  = "jpg"

#Storage Configuration
use_gcs_upload       = true
max_parallel_uploads = "20"  # 20 files uploaded concurrently
upload_chunk_size_mb = "16"  # 16MB chunk size


# Thumbnail Configuration
thumbnail_size    = 256
thumbnail_quality = 90

# Timeout Configuration (minutes)
format_conversion_timeout_minute = 20
dzi_conversion_timeout_minute    = 120

# Terraform State Bucket
tf_state_bucket = "your-tf-state-bucket"