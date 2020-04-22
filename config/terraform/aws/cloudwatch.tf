resource "aws_cloudwatch_log_group" "covidshield" {
  name = var.cloudwatch_log_group_name

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}
