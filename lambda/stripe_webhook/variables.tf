variable "environment" {
  type        = string
  description = "Environment name"
}

variable "region" {
  type        = string
  description = "AWS region"
}

variable "account_id" {
  type        = string
  description = "AWS account ID"
}

variable "payments_table_name" {
  type        = string
  description = "Name of the DynamoDB payments table"
}

variable "payments_table_arn" {
  type        = string
  description = "ARN of the DynamoDB payments table"
}

variable "lambda_artifacts_bucket" {
  type        = string
  description = "S3 bucket name for Lambda artifacts"
}

variable "lambda_artifacts_key" {
  type        = string
  description = "S3 key for the Lambda zip file"
}

variable "lambda_source_code_hash" {
  type        = string
  description = "Base64-encoded SHA256 hash of the Lambda zip file"
  default     = ""
}

variable "ssm_read_policy_arn" {
  type        = string
  description = "ARN of the SSM read policy"
}

variable "event_bus_name" {
  type        = string
  description = "Name of the EventBridge event bus"
  default     = "default"
}
