###
# AWS S3 bucket - WAF log target
###

resource "aws_s3_bucket" "exposure_config" {
  bucket = "covid-shield-exposure-config-${var.environment}"
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }

  logging {
    target_bucket = aws_s3_bucket.exposure_config_logs.bucket
  }
}

resource "aws_s3_bucket" "firehose_waf_logs" {
  bucket = "covid-shield-${var.environment}-waf-logs"
  acl    = "private"
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }
  lifecycle_rule {
    enabled = true

    expiration {
      days = 90
    }
  }
  #tfsec:ignore:AWS002 - Ignore log of logs
}

###
# AWS S3 bucket - cloudfront log target
###
resource "aws_s3_bucket" "cloudfront_logs" {
  bucket = "covid-shield-${var.environment}-cloudfront-logs"
  acl    = "private"
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }
  lifecycle_rule {
    enabled = true

    expiration {
      days = 90
    }
  }
  #tfsec:ignore:AWS002 - Ignore log of logs
}

resource "aws_s3_bucket" "exposure_config_logs" {
  bucket = "covid-shield-exposure-config-${var.environment}-logs"
  acl    = "log-delivery-write"
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }

  lifecycle_rule {
    enabled = true

    expiration {
      days = 90
    }
  }
  #tfsec:ignore:AWS002 - Ignore log of logs
}

resource "aws_s3_bucket_public_access_block" "firehose_waf_logs" {
  bucket = aws_s3_bucket.firehose_waf_logs.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_public_access_block" "cloudfront_logs" {
  bucket = aws_s3_bucket.cloudfront_logs.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_public_access_block" "exposure_config_logs" {
  bucket = aws_s3_bucket.exposure_config_logs.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}



