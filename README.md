# Payment Service

AWS-based payment service infrastructure using Terraform, Lambda, API Gateway, and DynamoDB.

## Overview

This service provides payment processing functionality using AWS serverless architecture.

## Architecture

- **API Gateway**: REST API endpoints with API key authentication
- **Lambda Functions**: Serverless compute for payment processing
- **DynamoDB**: NoSQL database for payment records
- **Terraform**: Infrastructure as Code for AWS resources
- **Terraform Cloud**: Remote execution and state management with AWS OIDC
- **GitHub Actions**: CI/CD trigger for Terraform Cloud runs

## Features

- Create payment links
- Process payments
- Webhook handling for payment events
- Secure payment data storage
- API key authentication
- Automated deployments via GitHub Actions

## Prerequisites

- AWS Account with appropriate permissions
- Terraform Cloud account and workspace configured with:
  - AWS OIDC provider for authentication
  - Remote execution mode enabled
  - AWS credentials configured in workspace
- GitHub repository
- Terraform >= 1.5.0 (for local development)
- Python 3.11+

## GitHub Secrets Required

Configure these secrets in your GitHub repository (Settings → Secrets and variables → Actions):

- `TF_API_TOKEN` - Terraform Cloud API token
- `TFC_WORKSPACE_ID` - Terraform Cloud workspace ID

## Setup

### 1. Configure Terraform Cloud

1. Create a Terraform Cloud workspace
2. Set execution mode to **Remote**
3. Configure AWS OIDC authentication in workspace
4. Set workspace variables:
   - `environment` = `dev`
   - `region` = `ap-southeast-1`

### 2. Configure GitHub Secrets

Add the following secrets to your GitHub repository:
- `TF_API_TOKEN` - Your Terraform Cloud API token
- `TFC_WORKSPACE_ID` - Your workspace ID (found in workspace settings)

### 3. Deploy Infrastructure

Push to the `dev` branch to trigger automatic deployment via Terraform Cloud:

```bash
git push origin dev
```

### 4. Local Development (Optional)

```bash
# Install pre-commit hooks
pip install pre-commit
pre-commit install

# Install Terraform tools
curl -s https://raw.githubusercontent.com/terraform-linters/tflint/master/install_linux.sh | bash

# Initialize Terraform
terraform init

# Plan changes (executed remotely in Terraform Cloud)
terraform plan
```

## Project Structure

```
.
├── api_gateway/          # API Gateway configuration
├── dynamodb/             # DynamoDB table definitions
├── lambda/               # Lambda function code and configuration
├── temp/                 # Temporary files and utilities
├── .pre-commit-config.yaml  # Pre-commit hooks configuration
├── .tflint.hcl          # TFLint configuration
├── main.tf              # Main Terraform configuration
├── variables.tf         # Terraform variables
├── outputs.tf           # Terraform outputs
└── README.md            # This file
```

## Development

### Pre-commit Hooks

This project uses Git pre-commit hooks to ensure code quality and consistent commit messages.

#### Automated Checks

**Python:**
- Black (code formatting)
- Flake8 (style guide enforcement)
- isort (import sorting)
- Pylint (code analysis)
- Bandit (security scanning)

**Terraform:**
- terraform fmt (formatting)
- terraform validate (validation)
- terraform-docs (documentation)
- tflint (linting with AWS rules)

**General:**
- Trailing whitespace removal
- End-of-file fixer
- YAML validation
- Large file detection
- Merge conflict detection
- Private key detection

#### Commit Message Format

Follows Conventional Commits specification:

```
<type>(<scope>): <subject>

Examples:
feat(api): add payment endpoint
fix(lambda): resolve timeout issue
docs: update README
```

**Valid types**: feat, fix, docs, style, refactor, test, chore, perf, ci, build, revert

#### Setup Pre-commit Hooks

```bash
# Install pre-commit
pip install pre-commit
pre-commit install

# Install Terraform tools
# TFLint
curl -s https://raw.githubusercontent.com/terraform-linters/tflint/master/install_linux.sh | bash

# Terraform-docs
curl -Lo ./terraform-docs.tar.gz https://github.com/terraform-docs/terraform-docs/releases/download/v0.17.0/terraform-docs-v0.17.0-$(uname)-amd64.tar.gz
tar -xzf terraform-docs.tar.gz
chmod +x terraform-docs
sudo mv terraform-docs /usr/local/bin/
rm terraform-docs.tar.gz

# Verifyd
pre-commit --version
terraform-docs --version
tflint --version
```

#### Usage

```bash
# Automatic on commit
git commit -m "feat(api): add new endpoint"

# Manual run
pre-commit run --all-files

# Skip hooks (not recommended)
git commit --no-verify -m "your message"
```

#### Configuration Files

- `.pre-commit-config.yaml` - Pre-commit hooks configuration
- `pyproject.toml` - Python tools configuration
- `.tflint.hcl` - Terraform linting rules
- `.git/hooks/commit-msg` - Commit message validator

## Documentation

- **[TERRAFORM.md](TERRAFORM.md)**: Auto-generated Terraform documentation

## API Endpoints

- `POST /payments` - Create a new payment

## Environment Variables

- `ENVIRONMENT` - Deployment environment (dev/staging/prod)
- `PAYMENTS_TABLE_NAME` - DynamoDB table name for payments
- `LOG_LEVEL` - Logging level (INFO/DEBUG/ERROR)

## License

[Add your license here]
