resource "aws_sns_topic" "alert_warning" {
  name       = "alert-warning"
  kms_key_id = aws_kms_key.cw.arn

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_sns_topic" "alert_critical" {
  name       = "alert-critical"
  kms_key_id = aws_kms_key.cw.arn

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}