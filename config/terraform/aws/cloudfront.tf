###
# AWS Cloudfront (CDN) - Key Retrieval - retrieval.{$route53_zone_name}
###

resource "aws_cloudfront_distribution" "key_server_distribution" {
  origin {
    domain_name = aws_lb.covidshield_key_server.dns_name
    origin_id   = aws_lb.covidshield_key_server.name

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  enabled         = true
  is_ipv6_enabled = true
  web_acl_id      = aws_wafv2_web_acl.key_retrieval_cdn.arn

  aliases = [
    "${var.route53_zone_name}",
    "retrieval.${var.route53_zone_name}",
    "submission.${var.route53_zone_name}"
  ]

  default_cache_behavior {
    allowed_methods  = ["HEAD", "DELETE", "POST", "GET", "OPTIONS", "PUT", "PATCH"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = aws_lb.covidshield_key_server.name

    forwarded_values {
      query_string = true
      headers      = ["Host", "authorization", "User-Agent"]

      cookies {
        forward = "all"
      }
    }

    viewer_protocol_policy = "https-only"
    min_ttl                = 0
    default_ttl            = 3600
    max_ttl                = 7200
    compress               = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.covidshield.certificate_arn
    minimum_protocol_version = "TLSv1.2_2019"
    ssl_support_method       = "sni-only"
  }

  depends_on = [
    aws_acm_certificate_validation.covidshield
  ]
}
