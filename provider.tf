provider "aws" {
  region = var.region

  default_tags {
    tags = {
      Terraform   = "true"
      Name        = "payment-service"
      Environment = var.environment
    }
  }
}
