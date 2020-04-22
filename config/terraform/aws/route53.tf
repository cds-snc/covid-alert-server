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
# Route53 Record - Key Retrieval
###

resource "aws_route53_record" "covidshield_key_retrieval" {
  zone_id = aws_route53_zone.covidshield.zone_id
  name    = "retrieval.${aws_route53_zone.covidshield.name}"
  type    = "A"

  alias {
    name                   = aws_lb.covidshield_key_retrieval.dns_name
    zone_id                = aws_lb.covidshield_key_retrieval.zone_id
    evaluate_target_health = false
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
    name                   = aws_lb.covidshield_key_submission.dns_name
    zone_id                = aws_lb.covidshield_key_submission.zone_id
    evaluate_target_health = false
  }
}
