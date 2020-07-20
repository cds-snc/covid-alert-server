###
# Route53 Zone
###

resource "aws_route53_zone" "covidshield" {
  name = var.route53_zone_name

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

###
# Route53 Record - Root Domain
###

resource "aws_route53_record" "covidshield_key_server" {
  zone_id = aws_route53_zone.covidshield.zone_id
  name    = aws_route53_zone.covidshield.name
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.key_server_distribution.domain_name
    zone_id                = aws_cloudfront_distribution.key_server_distribution.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_health_check" "covidshield_key_server_healthcheck" {
  fqdn              = aws_route53_zone.covidshield.name
  port              = 443
  type              = "HTTPS"
  resource_path     = "/services/ping"
  failure_threshold = "3"
  request_interval  = "30"

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

###
# Route53 Record - Key Retrieval
###

resource "aws_route53_record" "covidshield_key_retrieval" {
  zone_id = aws_route53_zone.covidshield.zone_id
  name    = "retrieval.${aws_route53_zone.covidshield.name}"
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.key_server_distribution.domain_name
    zone_id                = aws_cloudfront_distribution.key_server_distribution.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_health_check" "covidshield_key_retrieval_healthcheck" {
  fqdn              = "retrieval.${aws_route53_zone.covidshield.name}"
  port              = 443
  type              = "HTTPS"
  resource_path     = "/services/ping"
  failure_threshold = "3"
  request_interval  = "30"

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

###
# Route53 Record - Key Submission
###

resource "aws_route53_record" "covidshield_key_submission" {
  zone_id = aws_route53_zone.covidshield.zone_id
  name    = "submission.${aws_route53_zone.covidshield.name}"
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.key_server_distribution.domain_name
    zone_id                = aws_cloudfront_distribution.key_server_distribution.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_health_check" "covidshield_key_submission_healthcheck" {
  fqdn              = "submission.${aws_route53_zone.covidshield.name}"
  port              = 443
  type              = "HTTPS"
  resource_path     = "/services/ping"
  failure_threshold = "3"
  request_interval  = "30"

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}
