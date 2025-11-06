terraform {
  cloud {
    organization = "md-imam-hasan-org"

    workspaces {
      name = "serverless-architecture-with-terraform"
    }
  }

  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
