#!/bin/bash
# cleanup_state.sh - Import existing Cloud Run resources into Terraform state
# Prevents 409 Conflict errors when resources exist in GCP but not in TF state

set -euo pipefail

ENVIRONMENT="${1:-prod}"
echo "🔍 Checking Terraform state for environment: ${ENVIRONMENT}"

# Get project and region from Terraform state
PROJECT_ID=$(terraform output -raw -state=<(terraform show -json | jq -r '.values.root_module.resources[0].values.project // empty') 2>/dev/null || gcloud config get-value project)
REGION=$(gcloud config get-value compute/region 2>/dev/null || echo "")

# Try to get region from terraform state or gcloud
if [ -z "$REGION" ]; then
  REGION=$(terraform show -json 2>/dev/null | jq -r '.values.root_module.resources[0].values.location // empty' 2>/dev/null || echo "")
fi

if [ -z "$REGION" ]; then
  echo "❌ Could not determine region. Please set compute/region in gcloud config."
  exit 1
fi

echo "📦 Project: ${PROJECT_ID}"
echo "🌍 Region: ${REGION}"

WORKER_TYPES=("small" "medium" "large")

for WORKER_TYPE in "${WORKER_TYPES[@]}"; do
  # Determine job name based on environment
  if [ "$ENVIRONMENT" == "prod" ]; then
    JOB_NAME="image-processing-job-${WORKER_TYPE}"
  else
    JOB_NAME="image-processing-job-${WORKER_TYPE}-${ENVIRONMENT}"
  fi

  TF_RESOURCE="google_cloud_run_v2_job.image_processing_job[\"${WORKER_TYPE}\"]"

  # Check if resource is already in Terraform state
  if terraform state show "${TF_RESOURCE}" &>/dev/null; then
    echo "✅ ${JOB_NAME} already in Terraform state"
  else
    # Check if the job exists in GCP
    if gcloud run jobs describe "${JOB_NAME}" --region="${REGION}" --project="${PROJECT_ID}" &>/dev/null; then
      echo "📥 Importing ${JOB_NAME} into Terraform state..."
      terraform import \
        -var="image_tag=placeholder" \
        -var="environment=${ENVIRONMENT}" \
        -var="tf_state_bucket=${TF_STATE_BUCKET:-}" \
        "${TF_RESOURCE}" \
        "projects/${PROJECT_ID}/locations/${REGION}/jobs/${JOB_NAME}" || true
      echo "✅ Imported ${JOB_NAME}"
    else
      echo "ℹ️  ${JOB_NAME} does not exist in GCP (will be created)"
    fi
  fi
done

# Handle IAM member resources
for WORKER_TYPE in "${WORKER_TYPES[@]}"; do
  TF_IAM_RESOURCE="google_cloud_run_v2_job_iam_member.executor[\"${WORKER_TYPE}\"]"

  if terraform state show "${TF_IAM_RESOURCE}" &>/dev/null; then
    echo "✅ IAM binding for ${WORKER_TYPE} already in Terraform state"
  else
    echo "ℹ️  IAM binding for ${WORKER_TYPE} not in state (will be created/imported on apply)"
  fi
done

echo ""
echo "🏁 State cleanup complete!"
