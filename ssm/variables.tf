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

variable "kms_key_arn" {
  type        = string
  description = "KMS key ARN for decrypting SecureString parameters (optional)"
  default     = ""
}
