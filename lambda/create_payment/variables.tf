variable "environment" {
  type        = string
  description = "Environment name"
}

variable "payments_table_name" {
  type        = string
  description = "Name of the DynamoDB payments table"
}

variable "payments_table_arn" {
  type        = string
  description = "ARN of the DynamoDB payments table"
}
