resource "aws_kms_key" "cw" {
  description         = "CloudWatch Log Group Key"
  enable_key_rotation = true
}
