# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

### Local Development
```bash
# Initialize Terraform
terraform init

# Run pre-commit hooks manually
pre-commit run --all-files

# Build Lambda function locally (Go)
cd lambda/create_payment
make build

# Run Go tests
cd lambda/create_payment/src
go test -v ./...

# Run Go linting
cd lambda/create_payment/src
golangci-lint run

# Terraform validation (locally)
terraform validate
terraform fmt
tflint
```

### Deployment
Deployment is automated via GitHub Actions when pushing to `dev` branch:
```bash
git push origin dev
```

## Project Architecture

This is a serverless payment processing service built on AWS with the following key components:

### Infrastructure Structure
- **Terraform Cloud**: Remote state management and execution with AWS OIDC authentication
- **Modular Design**: Infrastructure organized into separate modules:
  - `api_gateway/`: REST API configuration with payments resource
  - `dynamodb/payments/`: Payment records table
  - `lambda/create_payment/`: Go-based Lambda function for payment processing

### Lambda Functions
- **Language**: Go 1.21+
- **Runtime**: Custom runtime using `bootstrap` binary
- **Architecture**: ARM64 (default) with AMD64 support
- **Build Process**: GitHub Actions builds binary and commits to repo

### CI/CD Pipeline
1. **Trigger**: Push to `dev` branch
2. **Build**: Go Lambda binary compiled for Linux ARM64
3. **Commit**: Binary committed to repository (`[skip ci]`)
4. **Deploy**: Terraform Cloud run triggered via API
5. **Monitor**: Workflow polls Terraform Cloud until completion

### Key Files
- `backend.tf`: Terraform Cloud remote backend configuration
- `provider.tf`: AWS provider with OIDC authentication
- `.github/workflows/deploy-dev.yml`: CI/CD pipeline
- `.pre-commit-config.yaml`: Development quality gates

### Environment Variables
- `ENVIRONMENT`: Deployment environment (dev/staging/prod)
- `PAYMENTS_TABLE_NAME`: DynamoDB table for payment records
- `LOG_LEVEL`: Logging configuration

### Security
- API Gateway with API key authentication
- AWS OIDC authentication for Terraform Cloud
- Pre-commit hooks for secret detection (gitleaks)
- Private key detection in pre-commit

## Development Workflow

1. **Setup**: Install Go 1.21+, Terraform, and pre-commit hooks
2. **Make Changes**: Edit Go code or Terraform configuration
3. **Local Validation**: Pre-commit hooks run automatically on commit
4. **Push**: `git push origin dev` triggers automated deployment
5. **Monitor**: GitHub Actions reports deployment status

## Module Dependencies
- API Gateway module depends on Lambda function
- Lambda function depends on DynamoDB table
- All resources use common variables from root module
