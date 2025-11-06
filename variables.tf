variable "region" {
  description = "region for all environments"
  default     = "ap-southeast-1"
}

variable "aws_role_arn" {
  description = "Role to assume for deploying resources"
  type        = string
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
