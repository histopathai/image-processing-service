# Environment (prod or dev)
environment = "prod"

# Docker image tag to deploy
image_tag = "latest"

# Logging
log_level = "INFO"

# DZI Configuration
tile_size   = 256
overlap     = 0
quality     = 85
dzi_layout  = "dz"
dzi_suffix  = "jpg"

# Thumbnail Configuration
thumbnail_size    = 256
thumbnail_quality = 90

# Timeout Configuration (minutes)
format_conversion_timeout_minute = 20
dzi_conversion_timeout_minute    = 120

# Terraform State Bucket
tf_state_bucket = "your-tf-state-bucket"