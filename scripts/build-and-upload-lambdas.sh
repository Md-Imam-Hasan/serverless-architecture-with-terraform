#!/bin/bash
set -e

# Configuration
S3_BUCKET="${LAMBDA_ARTIFACTS_BUCKET:-dev-lambda-artifacts-776724785158}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_ALL="${BUILD_ALL:-false}"

# Lambda directories and S3 keys
declare -A LAMBDAS=(
    ["create_payment"]="create_payment/lambda.zip"
    ["step_handler"]="step_handler/lambda.zip"
    ["stripe_webhook"]="stripe_webhook/lambda.zip"
)

# Detect changed lambdas
get_changed_lambdas() {
    local changed=()

    # Get changed files compared to previous commit
    local changed_files
    changed_files=$(git diff --name-only HEAD~1 HEAD 2>/dev/null || echo "")

    # Also check for changes in shared code
    local shared_changed=false
    if echo "$changed_files" | grep -q "^lambda/shared/"; then
        shared_changed=true
    fi

    for lambda_name in "${!LAMBDAS[@]}"; do
        # Check if this lambda's code changed or shared code changed
        if echo "$changed_files" | grep -q "^lambda/$lambda_name/" || [ "$shared_changed" = true ]; then
            changed+=("$lambda_name")
        fi
    done

    echo "${changed[@]}"
}

echo "=========================================="
echo "Building and Uploading Lambda Functions"
echo "=========================================="
echo "S3 Bucket: $S3_BUCKET"
echo ""

# Determine which lambdas to build
if [ "$BUILD_ALL" = "true" ]; then
    LAMBDAS_TO_BUILD=("${!LAMBDAS[@]}")
    echo "Mode: Building ALL lambdas (BUILD_ALL=true)"
else
    read -ra LAMBDAS_TO_BUILD <<< "$(get_changed_lambdas)"
    if [ ${#LAMBDAS_TO_BUILD[@]} -eq 0 ]; then
        echo "No lambda changes detected. Nothing to build."
        echo "Set BUILD_ALL=true to force build all lambdas."
        exit 0
    fi
    echo "Mode: Building CHANGED lambdas only"
    echo "Changed: ${LAMBDAS_TO_BUILD[*]}"
fi
echo ""

# Build and upload each Lambda
for lambda_name in "${LAMBDAS_TO_BUILD[@]}"; do
    s3_key="${LAMBDAS[$lambda_name]}"
    src_dir="$PROJECT_ROOT/lambda/$lambda_name/src"

    echo "----------------------------------------"
    echo "Building: $lambda_name"
    echo "----------------------------------------"

    if [ ! -d "$src_dir" ]; then
        echo "ERROR: Source directory not found: $src_dir"
        exit 1
    fi

    cd "$src_dir"

    # Clean previous build artifacts
    rm -f bootstrap lambda.zip

    # Build for AWS Lambda (Linux ARM64)
    echo "Compiling Go binary..."
    GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap .

    # Create zip
    echo "Creating zip archive..."
    zip -j lambda.zip bootstrap

    # Calculate hash for Terraform
    HASH=$(openssl dgst -sha256 -binary lambda.zip | openssl enc -base64)
    echo "Source code hash: $HASH"

    # Upload to S3
    echo "Uploading to s3://$S3_BUCKET/$s3_key..."
    aws s3 cp lambda.zip "s3://$S3_BUCKET/$s3_key"

    # Clean up
    rm -f bootstrap lambda.zip

    echo "✓ $lambda_name uploaded successfully"
    echo ""
done

echo "=========================================="
echo "All Lambda functions built and uploaded!"
echo "=========================================="
echo ""
echo "You can now run: terraform apply"
