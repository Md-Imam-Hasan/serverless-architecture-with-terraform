variable "environment" {
  type        = string
  description = "Environment name"
}

variable "step_handler_lambda_arn" {
  type        = string
  description = "ARN of the step handler Lambda function"
}
