provider "aws" {
  region = var.region

  assume_role {
    role_arn = var.aws_role_arn
  }

  default_tags {
    tags = {
      Terraform   = "true"
      Name        = "payment-service"
      Environment = var.environment
    }
  }
}
