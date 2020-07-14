###
# AWS S3 bucket - S3 bucket access logs
###
resource "aws_s3_bucket" "s3_access_logs" {
  bucket = "covid-shield-${var.environment}-s3-access-logs"
  acl    = "log-delivery-write"
  #tfsec:ignore:AWS002
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }
}

###
# AWS S3 bucket - WAF log target
###
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
  logging {
    target_bucket = aws_s3_bucket.s3_access_logs.id
    target_prefix = "waf/"
  }
}
