variable "environment" {
  type        = string
  description = "release environment name"
}

variable "api_acm_arn" {
  type = string
}

variable "create_payment_lambda_name" {
  type = string
}

variable "create_payment_lambda_invoke_arn" {
  type = string
}

variable "stripe_webhook_lambda_name" {
  type = string
}

variable "stripe_webhook_lambda_invoke_arn" {
  type = string
}
