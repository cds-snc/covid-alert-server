resource "aws_flow_log" "example" {
  log_destination      = aws_s3_bucket.vpc_flow_log.arn
  log_destination_type = "s3"
  traffic_type         = "ALL"
  vpc_id               = aws_vpc.covidshield.id
}

resource "aws_s3_bucket" "vpc_flow_log" {
  bucket = "${var.vpc_name}_flow_log_${random_string.random.result}"
  acl    = "private"
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }
}