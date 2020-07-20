# Enable shield on cloudfront distribution
resource "aws_shield_protection" "key_retrieval_distribution" {
  name         = "key_retrieval_distribution"
  resource_arn = aws_cloudfront_distribution.key_server_distribution.arn
}

# Enable shield on Route53 hosted zone
resource "aws_shield_protection" "route53_covidshield" {
  name = "route53_covidshield"
  //   resource_arn = aws_route53_zone.covidshield.arn
  resource_arn = "arn:aws:route53:::hostedzone/${aws_route53_zone.covidshield.zone_id}"
}

# Enable shield on ALBs
resource "aws_shield_protection" "alb_covidshield_key_server" {
  name         = "alb_covidshield_key_server"
  resource_arn = aws_lb.covidshield_key_server.arn
}
