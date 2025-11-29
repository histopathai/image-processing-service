output "job_names" {
  description = "Names of the deployed Cloud Run Jobs"
  value = {
    for k, v in google_cloud_run_v2_job.image_processing_job : k => v.name
  }
}

output "job_ids" {
  description = "Full resource IDs of the deployed Cloud Run Jobs"
  value = {
    for k, v in google_cloud_run_v2_job.image_processing_job : k => v.id
  }
}

output "job_uris" {
  description = "URIs of the deployed Cloud Run Jobs"
  value = {
    for k, v in google_cloud_run_v2_job.image_processing_job : k => "https://console.cloud.google.com/run/jobs/details/${v.location}/${v.name}"
  }
}