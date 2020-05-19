provider "aws" {
  version = "~> 2.0"
  region  = var.region
}

provider "github" {
  organization = "CovidShield"
  anonymous    = true
}

terraform {
  required_version = "> 0.12.0"
}

terraform {
  backend "s3" {
    bucket = "covidshield-terraform"
    key    = "aws/backend/default.tfstate"
    region = "ca-central-1"
  }
}
