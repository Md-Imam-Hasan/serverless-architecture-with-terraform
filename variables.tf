variable "region" {
  description = "region for all environments"
  default     = "ap-southeast-1"
}

variable "environment" {
  type        = string
  default     = "dev"
  description = "release environment name"
}
variable "account" {
  description = "account ID for all environment"
  default = {
    # "dev"     = "114290458322",
    "dev" = "776724785158",
    # "test"    = "885851041714",
    # "staging" = "475073270640",
    # "prod"    = "986575127476"
  }
}

variable "lambda_artifacts_key" {
  type        = string
  description = "S3 key for the create_payment Lambda zip file"
  default     = "create_payment/lambda.zip"
}

variable "lambda_source_code_hash" {
  type        = string
  description = "Base64-encoded SHA256 hash of the create_payment Lambda zip file"
  default     = ""
}

variable "step_handler_lambda_artifacts_key" {
  type        = string
  description = "S3 key for the step_handler Lambda zip file"
  default     = "step_handler/lambda.zip"
}

variable "step_handler_lambda_source_code_hash" {
  type        = string
  description = "Base64-encoded SHA256 hash of the step_handler Lambda zip file"
  default     = ""
}

variable "stripe_webhook_lambda_artifacts_key" {
  type        = string
  description = "S3 key for the stripe_webhook Lambda zip file"
  default     = "stripe_webhook/lambda.zip"
}

variable "stripe_webhook_lambda_source_code_hash" {
  type        = string
  description = "Base64-encoded SHA256 hash of the stripe_webhook Lambda zip file"
  default     = ""
}

variable "event_bus_name" {
  type        = string
  description = "Name of the EventBridge event bus"
  default     = "default"
}
