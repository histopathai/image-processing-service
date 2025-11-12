variable "environment" {
  description = "Environment name (prod, dev)"
  type        = string

  validation {
    condition     = contains(["prod", "dev"], var.environment)
    error_message = "Environment must be either 'prod' or 'dev'."
  }
}

variable "min_instances" {
  description = "Minimum number of instances for scaling"
  type        = number
  default     = 0
}

variable "max_instances" {
  description = "Maximum number of instances for scaling"
  type        = number
  default     = 10
}

variable "cpu_limit" {
  description = "CPU limit for each instance"
  type        = string
  default     = "2"
}

variable "memory_limit" {
  description = "Memory limit for each instance"
  type        = string
  default     = "8Gi"
}

variable "allow_public_access" {
  description = "Allow public access to the service"
  type        = bool
  default     = false
}

variable "log_level" {
  description = "Log level (DEBUG, INFO, WARN, ERROR)"
  type        = string
  default     = "INFO"

  validation {
    condition     = contains(["DEBUG", "INFO", "WARN", "ERROR"], var.log_level)
    error_message = "Log level must be one of 'DEBUG', 'INFO', 'WARN', or 'ERROR'."
  }
}

variable "image_tag" {
  description = "Docker image tag to deploy"
  type        = string
}

variable "tile_size" {
  description = "DZI tile size"
  type        = number
  default     = 256
}

variable "overlap" {
  description = "DZI tile overlap"
  type        = number
  default     = 0
}

variable "quality" {
  description = "DZI tile quality"
  type        = number
  default     = 85
}

variable "dzi_layout" {
  description = "DZI layout"
  type        = string
  default     = "dz"
}

variable "dzi_suffix" {
  description = "DZI suffix"
  type        = string
  default     = "jpg"
}

variable "thumbnail_size" {
  description = "Thumbnail size"
  type        = number
  default     = 256
}

variable "thumbnail_quality" {
  description = "Thumbnail quality"
  type        = number
  default     = 90
}