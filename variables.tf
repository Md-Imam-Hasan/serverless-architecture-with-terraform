variable "region" {
  description = "region for all environments"
  default = "eu-west-1"
}
variable "environment" {
  type        = string
  description = "release environment name"
}
variable "account" {
  description = "account ID for all environment"
  default = {
    "dev"     = "114290458322",
    "test"    = "885851041714",
    "staging" = "475073270640",
    "prod"    = "986575127476"
  }
}