#!/bin/bash
set -e

# Configuration
S3_BUCKET="${LAMBDA_ARTIFACTS_BUCKET:-dev-lambda-artifacts-776724785158}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Lambda directories and S3 keys
declare -A LAMBDAS=(
    ["create_payment"]="create_payment/lambda.zip"
    ["step_handler"]="step_handler/lambda.zip"
    ["stripe_webhook"]="stripe_webhook/lambda.zip"
)

echo "=========================================="
echo "Building and Uploading Lambda Functions"
echo "=========================================="
echo "S3 Bucket: $S3_BUCKET"
echo ""

# Build all lambdas
LAMBDAS_TO_BUILD=("${!LAMBDAS[@]}")
echo "Building: ${LAMBDAS_TO_BUILD[*]}"
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
