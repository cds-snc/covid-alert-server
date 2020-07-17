###
# AWS IPSet - list of IPs/CIDRs to allow
###
resource "aws_wafv2_ip_set" "new_key_claim" {
  name               = "new-key-claim"
  description        = "New Key Claim Allow IPs/CIDRs"
  scope              = "REGIONAL"
  ip_address_version = "IPV4"
  addresses          = toset(var.new_key_claim_allow_list)
}

###
# AWS WAF - Managed Rules
###
resource "aws_wafv2_web_acl" "key_submission" {
  name  = "key_submission"
  scope = "REGIONAL"

  default_action {
    block {}
  }

  rule {
    name     = "AWSManagedRulesAmazonIpReputationList"
    priority = 1

    override_action {
      count {}
    }

    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesAmazonIpReputationList"
        vendor_name = "AWS"
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSManagedRulesAmazonIpReputationList"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "AWSManagedRulesCommonRuleSet"
    priority = 2

    override_action {
      count {}
    }

    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesCommonRuleSet"
        vendor_name = "AWS"
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSManagedRulesCommonRuleSet"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "AWSManagedRulesKnownBadInputsRuleSet"
    priority = 3

    override_action {
      count {}
    }

    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesKnownBadInputsRuleSet"
        vendor_name = "AWS"
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSManagedRulesKnownBadInputsRuleSet"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "AWSManagedRulesLinuxRuleSet"
    priority = 4

    override_action {
      count {}
    }

    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesLinuxRuleSet"
        vendor_name = "AWS"
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSManagedRulesLinuxRuleSet"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "AWSManagedRulesSQLiRuleSet"
    priority = 5

    override_action {
      count {}
    }

    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesSQLiRuleSet"
        vendor_name = "AWS"
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSManagedRulesSQLiRuleSet"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "KeySubmissionClaimKeyURIRateLimit"
    priority = 100

    action {
      count {}
    }

    statement {
      rate_based_statement {
        limit              = 100
        aggregate_key_type = "IP"
        scope_down_statement {
          byte_match_statement {
            positional_constraint = "EXACTLY"
            field_to_match {
              uri_path {}
            }
            search_string = "/claim-key"
            text_transformation {
              priority = 1
              type     = "COMPRESS_WHITE_SPACE"
            }
            text_transformation {
              priority = 2
              type     = "LOWERCASE"
            }
          }
        }
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "KeySubmissionClaimKeyURIRateLimit"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "KeySubmissionURIs"
    priority = 200

    action {
      allow {}
    }

    statement {
      or_statement {
        statement {
          byte_match_statement {
            positional_constraint = "STARTS_WITH"
            field_to_match {
              uri_path {}
            }
            search_string = "/services/"
            text_transformation {
              priority = 1
              type     = "COMPRESS_WHITE_SPACE"
            }
            text_transformation {
              priority = 2
              type     = "LOWERCASE"
            }
          }
        }
        statement {
          byte_match_statement {
            positional_constraint = "STARTS_WITH"
            field_to_match {
              uri_path {}
            }
            search_string = "/exposure-configuration/"
            text_transformation {
              priority = 1
              type     = "COMPRESS_WHITE_SPACE"
            }
            text_transformation {
              priority = 2
              type     = "LOWERCASE"
            }
          }
        }
        statement {
          byte_match_statement {
            positional_constraint = "EXACTLY"
            field_to_match {
              uri_path {}
            }
            search_string = "/upload"
            text_transformation {
              priority = 1
              type     = "COMPRESS_WHITE_SPACE"
            }
            text_transformation {
              priority = 2
              type     = "LOWERCASE"
            }
          }
        }
        statement {
          byte_match_statement {
            positional_constraint = "EXACTLY"
            field_to_match {
              uri_path {}
            }
            search_string = "/claim-key"
            text_transformation {
              priority = 1
              type     = "COMPRESS_WHITE_SPACE"
            }
            text_transformation {
              priority = 2
              type     = "LOWERCASE"
            }
          }
        }
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "KeySubmissionURIs"
      sampled_requests_enabled   = false
    }
  }

  rule {
    name     = "NewKeyClaimURI"
    priority = 201

    action {
      allow {}
    }

    statement {
      and_statement {
        statement {
          byte_match_statement {
            positional_constraint = "STARTS_WITH"
            field_to_match {
              uri_path {}
            }
            search_string = "/new-key-claim"
            text_transformation {
              priority = 1
              type     = "COMPRESS_WHITE_SPACE"
            }
            text_transformation {
              priority = 2
              type     = "LOWERCASE"
            }
          }
        }
        statement {
          byte_match_statement {
            positional_constraint = "STARTS_WITH"
            field_to_match {
              single_header {
                name = "authorization"
              }
            }
            search_string = "Bearer"
            text_transformation {
              priority = 1
              type     = "NONE"
            }
          }
        }
        statement {
          ip_set_reference_statement {
            arn = aws_wafv2_ip_set.new_key_claim.arn
          }
        }
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "NewKeyClaimURI"
      sampled_requests_enabled   = false
    }
  }

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "key_submission"
    sampled_requests_enabled   = false
  }
}

resource "aws_wafv2_web_acl" "key_retrieval" {
  name  = "key_retrieval"
  scope = "REGIONAL"

  default_action {
    allow {}
  }
}

resource "aws_wafv2_web_acl" "key_retrieval_cdn" {
  provider = aws.us-east-1

  name  = "key_retrieval_cdn"
  scope = "CLOUDFRONT"

  default_action {
    allow {}
  }
}

###
# AWS WAF - Resource Assocation
###
resource "aws_wafv2_web_acl_association" "key_submission_assocation" {
  resource_arn = aws_lb.covidshield_key_submission.arn
  web_acl_arn  = aws_wafv2_web_acl.key_submission.arn
}

resource "aws_wafv2_web_acl_association" "key_retrieval_assocation" {
  resource_arn = aws_lb.covidshield_key_retrieval.arn
  web_acl_arn  = aws_wafv2_web_acl.key_retrieval.arn
}

resource "aws_wafv2_web_acl_association" "key_retrieval_cdn_assocation" {
  provider = aws.us-east-1

  resource_arn = aws_cloudfront_distribution.key_retrieval_distribution.arn
  web_acl_arn  = aws_wafv2_web_acl.key_retrieval_cdn.arn
}

###
# AWS WAF - Logging
###
resource "aws_wafv2_web_acl_logging_configuration" "firehose_waf_logs" {
  log_destination_configs = ["${aws_kinesis_firehose_delivery_stream.firehose_waf_logs.arn}"]
  resource_arn            = aws_wafv2_web_acl.key_submission.arn
  redacted_fields {
    single_header {
      name = "authorization"
    }
  }
}

resource "aws_wafv2_web_acl_logging_configuration" "firehose_waf_logs_retrieval" {
  log_destination_configs = ["${aws_kinesis_firehose_delivery_stream.firehose_waf_logs.arn}"]
  resource_arn            = aws_wafv2_web_acl.key_retrieval.arn
}

resource "aws_wafv2_web_acl_logging_configuration" "firehose_waf_logs_retrieval_cdn" {
  log_destination_configs = ["${aws_kinesis_firehose_delivery_stream.firehose_waf_logs.arn}"]
  resource_arn            = aws_wafv2_web_acl.key_retrieval_cdn.arn
}