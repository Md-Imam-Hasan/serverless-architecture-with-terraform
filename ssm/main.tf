# SSM Parameter Store for Payment Service
# Stores company-specific Stripe keys and configuration

# Example company configuration - you'll need to create these manually or via CI/CD
# Parameters are stored at:
# /payment-service/{env}/companies/{company_name}/stripe_keys (SecureString)
# /payment-service/{env}/companies/{company_name}/config (String)

# IAM Policy for Lambda to read SSM parameters
resource "aws_iam_policy" "ssm_read_policy" {
  name        = "${var.environment}_payment_service_ssm_read_policy"
  description = "Policy to allow reading SSM parameters for payment service"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParameters",
          "ssm:GetParametersByPath"
        ]
        Resource = [
          "arn:aws:ssm:${var.region}:${var.account_id}:parameter/payment-service/${var.environment}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt"
        ]
        Resource = [
          var.kms_key_arn != "" ? var.kms_key_arn : "arn:aws:kms:${var.region}:${var.account_id}:key/*"
        ]
        Condition = {
          StringEquals = {
            "kms:ViaService" = "ssm.${var.region}.amazonaws.com"
          }
        }
      }
    ]
  })
}

# SSM Parameters with placeholder values - update manually in AWS Console after creation
resource "aws_ssm_parameter" "example_company_stripe_keys" {
  name        = "/payment-service/${var.environment}/companies/example_company/stripe_keys"
  description = "Stripe API keys for example_company"
  type        = "SecureString"
  value = jsonencode({
    secret_key      = "sk_test_REPLACE_ME"
    publishable_key = "pk_test_REPLACE_ME"
    webhook_secret  = "whsec_REPLACE_ME"
  })

  tags = {
    Environment = var.environment
    Service     = "payment-service"
  }

  lifecycle {
    ignore_changes = [value]
  }
}

resource "aws_ssm_parameter" "example_company_config" {
  name        = "/payment-service/${var.environment}/companies/example_company/config"
  description = "Configuration for example_company"
  type        = "String"
  value = jsonencode({
    base_url    = "https://example.com"
    success_api = "/api/payment/success"
  })

  tags = {
    Environment = var.environment
    Service     = "payment-service"
  }

  lifecycle {
    ignore_changes = [value]
  }
}
