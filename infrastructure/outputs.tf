output "service_url" {
  description = "The URL of the deployed image processing service"
  value       = google_cloud_run_v2_service.image_processing_service.uri
}