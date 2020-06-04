provider "aws" {
  version = "~> 2.0"
  region  = var.region
}

provider "aws" {
  version = "~> 2.0"
  alias   = "us-east-1"
  region  = "us-east-1"
}

provider "github" {
  organization = "cds-snc"
  anonymous    = false
}

terraform {
  required_version = "> 0.12.0"
}

terraform {
  backend "s3" {}
}
