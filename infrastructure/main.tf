terraform {
  required_version = ">=1.5.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 5.0"
    }
  }
  backend "gcs" {
    prefix = "services/image-processing-service"
  }
}

data "terraform_remote_state" "platform" {
  backend = "gcs"

  config = {
    bucket = var.tf_state_bucket
    prefix = "platform/prod"
  }
}

locals {
  # GCP project and region info
  project_id     = data.terraform_remote_state.platform.outputs.project_id
  project_number = data.terraform_remote_state.platform.outputs.project_number
  region         = data.terraform_remote_state.platform.outputs.region

  # Service info
  service_account        = data.terraform_remote_state.platform.outputs.image_processing_service_account_email
  artifact_repository_id = data.terraform_remote_state.platform.outputs.artifact_repository_id
  image_name             = "${local.region}-docker.pkg.dev/${local.project_id}/${local.artifact_repository_id}/image-processing-service:${var.image_tag}"

  original_bucket_name  = data.terraform_remote_state.platform.outputs.original_bucket_name
  processed_bucket_name = data.terraform_remote_state.platform.outputs.processed_bucket_name

  input_mount_path  = "/gcs/${local.original_bucket_name}"
  output_mount_path = "/gcs/${local.processed_bucket_name}"

  # Job configurations
  job_configs = {
    small = {
      cpu         = "1"
      memory      = "4Gi"
      timeout     = "1800s" # 30 minutes
      max_retries = 3
    }
    medium = {
      cpu         = "2"
      memory      = "8Gi"
      timeout     = "7200s" # 120 minutes
      max_retries = 2
    }
    large = {
      cpu         = "4"
      memory      = "16Gi"
      timeout     = "7200s" # 120 minutes
      max_retries = 1
    }
  }
}

provider "google" {
  project = local.project_id
  region  = local.region
}

provider "google-beta" {
  project = local.project_id
  region  = local.region
}

# Create Cloud Run Jobs for each size
resource "google_cloud_run_v2_job" "image_processing_job" {
  provider = google-beta
  for_each = local.job_configs

  name     = var.environment == "prod" ? "image-processing-job-${each.key}" : "image-processing-job-${each.key}-${var.environment}"
  location = local.region

  template {
    template {
      service_account = local.service_account
      timeout         = each.value.timeout
      max_retries     = each.value.max_retries

      volumes {
        name = "input-bucket"
        gcs {
          bucket    = local.original_bucket_name
          read_only = true
        }
      }
      volumes {
        name = "output-bucket"
        gcs {
          bucket    = local.processed_bucket_name
          read_only = false
        }
      }

      containers {
        image = local.image_name

        resources {
          limits = {
            cpu    = each.value.cpu
            memory = each.value.memory
          }
        }

        volume_mounts {
          name       = "input-bucket"
          mount_path = local.input_mount_path
        }

        volume_mounts {
          name       = "output-bucket"
          mount_path = local.output_mount_path
        }

        # Base environment variables
        env {
          name  = "PROJECT_ID"
          value = local.project_id
        }
        env {
          name  = "PROJECT_NUMBER"
          value = local.project_number
        }
        env {
          name  = "REGION"
          value = local.region
        }
        env {
          name  = "APP_ENV"
          value = var.environment == "prod" ? "PROD" : "DEV"
        }
        env {
          name  = "WORKER_TYPE"
          value = each.key
        }
        env {
          name  = "LOG_LEVEL"
          value = var.log_level
        }
        env {
          name  = "LOG_FORMAT"
          value = "json"
        }
        env {
          name  = "ORIGINAL_BUCKET_NAME"
          value = local.original_bucket_name
        }
        env {
          name  = "PROCESSED_BUCKET_NAME"
          value = local.processed_bucket_name
        }
        env {
          name  = "IMAGE_PROCESS_RESULT_TOPIC_ID"
          value = data.terraform_remote_state.platform.outputs.processing_completed_topic
        }
        env {
          name  = "INPUT_MOUNT_PATH"
          value = local.input_mount_path
        }
        env {
          name  = "OUTPUT_MOUNT_PATH"
          value = local.output_mount_path
        }
        env {
          name  = "TILE_SIZE"
          value = var.tile_size
        }
        env {
          name  = "OVERLAP"
          value = var.overlap
        }
        env {
          name  = "QUALITY"
          value = var.quality
        }
        env {
          name  = "DZI_LAYOUT"
          value = var.dzi_layout
        }
        env {
          name  = "DZI_SUFFIX"
          value = var.dzi_suffix
        }
        env {
          name  = "THUMBNAIL_SIZE"
          value = var.thumbnail_size
        }
        env {
          name  = "THUMBNAIL_QUALITY"
          value = var.thumbnail_quality
        }
        env {
          name  = "FORMAT_CONVERSION_TIMEOUT_MINUTE"
          value = var.format_conversion_timeout_minute
        }
        env {
          name  = "DZI_CONVERSION_TIMEOUT_MINUTE"
          value = var.dzi_conversion_timeout_minute
        }
      }
    }
  }

  labels = {
    environment = var.environment
    service     = "image-processing-service"
    worker_type = each.key
    managed_by  = "terraform"
  }
}

# Grant the main service permission to execute the jobs
resource "google_cloud_run_v2_job_iam_member" "executor" {
  provider = google-beta
  for_each = google_cloud_run_v2_job.image_processing_job

  project  = each.value.project
  location = each.value.location
  name     = each.value.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${data.terraform_remote_state.platform.outputs.main_service_account_email}"
}