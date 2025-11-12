terraform {
  required_version = ">=1.5.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
  backend "gcs" {
    bucket = "tf-state-histopathai-platform"
    prefix = "services/image-processing-service"
  }
}

data "terraform_remote_state" "platform" {
  backend = "gcs"

  config = {
    bucket = "tf-state-histopathai-platform"
    prefix = "platform/prod"
  }
}

locals {
  project_id      = data.terraform_remote_state.platform.outputs.project_id
  project_number  = data.terraform_remote_state.platform.outputs.project_number
  region          = data.terraform_remote_state.platform.outputs.region
  service_account = data.terraform_remote_state.platform.outputs.image_processing_service_account_email

  artifact_repository_id = data.terraform_remote_state.platform.outputs.artifact_repository_id
  service_name = var.environment == "prod" ? "image-processing-service" : "image-processing-service-${var.environment}"
  image_name   = "${local.region}-docker.pkg.dev/${local.project_id}/${local.artifact_repository_id}/${local.service_name}:${var.image_tag}"

  original_bucket_name  = data.terraform_remote_state.platform.outputs.original_bucket_name
  processed_bucket_name = data.terraform_remote_state.platform.outputs.processed_bucket_name

  input_mount_path  = "/gcs/${local.original_bucket_name}"
  output_mount_path = "/gcs/${local.processed_bucket_name}"
}

provider "google" {
  project = local.project_id
  region  = local.region
}

resource "google_cloud_run_v2_service" "image_processing_service" {
  name     = local.service_name
  location = local.region
  ingress  = var.allow_public_access ? "INGRESS_TRAFFIC_ALL" : "INGRESS_TRAFFIC_INTERNAL_ONLY"

  template {
    service_account = local.service_account
    timeout         = "3600s"
    scaling {
      min_instance_count = var.min_instances
      max_instance_count = var.max_instances
    }

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
          cpu    = var.cpu_limit
          memory = var.memory_limit
        }
      }

      ports {
        container_port = 8080
      }

      volume_mounts {
        name       = "input-bucket"
        mount_path = local.input_mount_path
      }
      
      volume_mounts {
        name       = "output-bucket"
        mount_path = local.output_mount_path
      }

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
        name  = "IMAGE_PROCESSING_SUB_ID"
        value = data.terraform_remote_state.platform.outputs.image_processing_subscription
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
    }
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }

  labels = {
    environment = var.environment
    service     = "image-processing-service"
    managed_by  = "terraform"
  }
}

resource "google_cloud_run_v2_service_iam_member" "public_access" {
  count = var.allow_public_access ? 1 : 0

  project  = google_cloud_run_v2_service.image_processing_service.project
  location = google_cloud_run_v2_service.image_processing_service.location
  name     = google_cloud_run_v2_service.image_processing_service.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}