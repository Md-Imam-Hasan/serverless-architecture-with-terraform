variable "api_gateway_id" {
  type = string
}

variable "api_gateway_root_resource_id" {
  type = string
}

variable "api_gateway_execution_arn" {
  type = string
}

variable "request_body_validator_id" {
  type = string
}

variable "create_payment_lambda_name" {
  type    = string
  default = ""
}

variable "create_payment_lambda_invoke_arn" {
  type    = string
  default = ""
}
