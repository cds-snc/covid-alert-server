resource "aws_sns_topic" "alert_warning" {
  name              = "alert-warning"
  kms_master_key_id = aws_kms_key.cw.arn

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_sns_topic" "alert_critical" {
  name              = "alert-critical"
  kms_master_key_id = aws_kms_key.cw.arn

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_sns_topic" "alert_warning_us_east" {
  provider = aws.us-east-1

  name              = "alert-warning"
  kms_master_key_id = aws_kms_key.cw_us_east.arn

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_sns_topic" "alert_critical_us_east" {
  provider = aws.us-east-1

  name              = "alert-critical"
  kms_master_key_id = aws_kms_key.cw_us_east.arn

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}