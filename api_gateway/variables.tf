variable "environment" {
  type        = string
  description = "release environment name"
}

variable "api_acm_arn" {
  type    = string
  default = ""
}

variable "create_payment_lambda_name" {
  type    = string
  default = ""
}

variable "create_payment_lambda_invoke_arn" {
  type    = string
  default = ""
}