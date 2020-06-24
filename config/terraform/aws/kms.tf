resource "aws_kms_key" "cw" {
  description         = "CloudWatch Log Group Key"
  enable_key_rotation = true
}

resource "aws_kms_alias" "cw" {
  name          = "alias/cloudwatch"
  target_key_id = "aws_kms_key.cw.key_id"
}
