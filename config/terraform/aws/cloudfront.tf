###
# AWS Cloudfront (CDN) - Key Retrieval - retrieval.{$route53_zone_name}
###

resource "aws_cloudfront_origin_access_identity" "origin_access_identity" {
  comment = "cloudfront origin access identity"
}

resource "aws_cloudfront_distribution" "key_retrieval_distribution" {
  origin {
    domain_name = aws_lb.covidshield_key_retrieval.dns_name
    origin_id   = aws_lb.covidshield_key_retrieval.name


    custom_header {
      name  = "covidshield"
      value = var.cloudfront_custom_header
    }

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  origin {
    domain_name = aws_s3_bucket.exposure_config.bucket_regional_domain_name
    origin_id   = "covid-shield-exposure-config-${var.environment}"

    s3_origin_config {
      origin_access_identity = aws_cloudfront_origin_access_identity.origin_access_identity.cloudfront_access_identity_path
    }
  }


  enabled         = true
  is_ipv6_enabled = true
  web_acl_id      = aws_wafv2_web_acl.key_retrieval_cdn.arn

  aliases = ["retrieval.${var.route53_zone_name}"]

  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = aws_lb.covidshield_key_retrieval.name

    forwarded_values {
      query_string = true
      headers      = ["Host"]

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

  # ordered_cache_behavior {
  #  path_pattern     = "/exposure-configuration/*"
  #  allowed_methods  = ["GET", "HEAD"]
  #  cached_methods   = ["GET", "HEAD"]
  #  target_origin_id = "covid-shield-exposure-config-${var.environment}"

  #  forwarded_values {
  #    query_string = false
  #    headers      = ["Origin"]

  #    cookies {
  #      forward = "none"
  #    }
  #  }

  #  viewer_protocol_policy = "https-only"
  #  min_ttl                = 0
  #  default_ttl            = 86400
  #  max_ttl                = 31536000
  #  compress               = true
  # }


  price_class = "PriceClass_100"

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.retrieval_covidshield.certificate_arn
    minimum_protocol_version = "TLSv1.2_2018"
    ssl_support_method       = "sni-only"
  }

  logging_config {
    include_cookies = false
    bucket          = aws_s3_bucket.cloudfront_logs.bucket_domain_name
    prefix          = "cloudfront"
  }

  depends_on = [aws_s3_bucket.cloudfront_logs]
}
