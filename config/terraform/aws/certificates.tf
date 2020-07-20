resource "aws_acm_certificate" "submission_covidshield" {
  domain_name = "submission.${var.route53_zone_name}"
  subject_alternative_names = [
    "retrieval.${var.route53_zone_name}",
    var.route53_zone_name
  ]
  validation_method = "DNS"

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }

  lifecycle {
    create_before_destroy = true
  }
}

###
# Cloudfront requires client certificate to be created in us-east-1
###
resource "aws_acm_certificate" "covidshield" {
  provider                  = aws.us-east-1
  domain_name               = var.route53_zone_name
  subject_alternative_names = ["*.${var.route53_zone_name}"]
  validation_method         = "DNS"

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }

  lifecycle {
    create_before_destroy = true
  }

}

resource "aws_route53_record" "submission_covidshield_certificate_validation" {
  zone_id         = aws_route53_zone.covidshield.zone_id
  allow_overwrite = true
  name            = aws_acm_certificate.submission_covidshield.domain_validation_options.0.resource_record_name
  type            = aws_acm_certificate.submission_covidshield.domain_validation_options.0.resource_record_type
  records         = [aws_acm_certificate.submission_covidshield.domain_validation_options.0.resource_record_value]
  ttl             = 60
}

resource "aws_route53_record" "submission_covidshield_certificate_validation_alt1" {
  zone_id         = aws_route53_zone.covidshield.zone_id
  allow_overwrite = true
  name            = aws_acm_certificate.submission_covidshield.domain_validation_options.1.resource_record_name
  type            = aws_acm_certificate.submission_covidshield.domain_validation_options.1.resource_record_type
  records         = [aws_acm_certificate.submission_covidshield.domain_validation_options.1.resource_record_value]
  ttl             = 60
}

resource "aws_route53_record" "submission_covidshield_certificate_validation_alt2" {
  zone_id         = aws_route53_zone.covidshield.zone_id
  allow_overwrite = true
  name            = aws_acm_certificate.submission_covidshield.domain_validation_options.2.resource_record_name
  type            = aws_acm_certificate.submission_covidshield.domain_validation_options.2.resource_record_type
  records         = [aws_acm_certificate.submission_covidshield.domain_validation_options.2.resource_record_value]
  ttl             = 60
}

resource "aws_route53_record" "covidshield_certificate_validation" {
  provider        = aws.us-east-1
  allow_overwrite = true
  zone_id         = aws_route53_zone.covidshield.zone_id
  name            = aws_acm_certificate.covidshield.domain_validation_options.0.resource_record_name
  type            = aws_acm_certificate.covidshield.domain_validation_options.0.resource_record_type
  records         = [aws_acm_certificate.covidshield.domain_validation_options.0.resource_record_value]
  ttl             = 60
}

resource "aws_route53_record" "wildcard_covidshield_certificate_validation" {
  zone_id         = aws_route53_zone.covidshield.zone_id
  allow_overwrite = true
  name            = aws_acm_certificate.covidshield.domain_validation_options.1.resource_record_name
  type            = aws_acm_certificate.covidshield.domain_validation_options.1.resource_record_type
  records         = [aws_acm_certificate.covidshield.domain_validation_options.1.resource_record_value]
  ttl             = 60
}

resource "aws_acm_certificate_validation" "submission_covidshield" {
  certificate_arn = aws_acm_certificate.submission_covidshield.arn
  validation_record_fqdns = [
    aws_route53_record.submission_covidshield_certificate_validation.fqdn,
    aws_route53_record.submission_covidshield_certificate_validation_alt1.fqdn,
    aws_route53_record.submission_covidshield_certificate_validation_alt2.fqdn,
  ]
}

resource "aws_acm_certificate_validation" "covidshield" {
  provider        = aws.us-east-1
  certificate_arn = aws_acm_certificate.covidshield.arn
  validation_record_fqdns = [
    aws_route53_record.covidshield_certificate_validation.fqdn,
    aws_route53_record.wildcard_covidshield_certificate_validation.fqdn,
  ]
}
