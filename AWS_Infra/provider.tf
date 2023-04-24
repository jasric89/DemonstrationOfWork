terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
      
    }
  }
}

provider "aws" {
  region = "eu-west-1"
  shared_config_files      = ["/home/jason/Dev_Ops/demonstration_of_work_repo/DemonstrationOfWork/AWS_Infra/conf"]
  shared_credentials_files = ["/home/jason/Dev_Ops/demonstration_of_work_repo/DemonstrationOfWork/AWS_Infra/conf"]
  profile                  = "terraform"
}