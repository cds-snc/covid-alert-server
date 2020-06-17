resource "aws_sns_topic" "alert_warning" {
  name = "alert-warning"
}

resource "aws_sns_topic" "alert_critical" {
  name = "alert-critical"
}