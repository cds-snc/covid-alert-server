###
# AWS WAF - Key Submission - /claim-key
###

resource "aws_wafregional_byte_match_set" "key_submission_claim_key_uri" {
  name = "KeySubmissionClaimKeyURI"
  byte_match_tuples {
    text_transformation   = "NONE"
    target_string         = "/claim-key"
    positional_constraint = "CONTAINS"

    field_to_match {
      type = "URI"
    }
  }
}

resource "aws_wafregional_rate_based_rule" "key_submission_claim_key_uri" {
  name        = "KeySubmissionClaimKeyURIRateLimit"
  metric_name = "KeySubmissionClaimKeyURIRateLimit"
  rate_key    = "IP"

  rate_limit = 100

  predicate {
    type    = "ByteMatch"
    data_id = aws_wafregional_byte_match_set.key_submission_claim_key_uri.id
    negated = false
  }
}

###
# AWS WAF - Key Submission - /new-key-claim
###

resource "aws_wafregional_byte_match_set" "key_submission_new_key_claim_uri" {
  name = "KeySubmissionNewKeyClaimURI"
  byte_match_tuples {
    text_transformation   = "NONE"
    target_string         = "/new-key-claim"
    positional_constraint = "CONTAINS"

    field_to_match {
      type = "URI"
    }
  }
}

resource "aws_wafregional_byte_match_set" "key_submission_authorization_header" {
  name = "KeySubmissionAuthorizationHeader"
  byte_match_tuples {
    text_transformation   = "NONE"
    target_string         = "Bearer"
    positional_constraint = "CONTAINS"

    field_to_match {
      type = "HEADER"
      data = "authorization"
    }
  }
}

resource "aws_wafregional_rule" "key_submission_new_key_claim_authorization_header" {
  name        = "KeySubmissionNewKeyClaimURIMissingAuthorizationHeader"
  metric_name = "KeySubmissionNewKeyClaimURIMissingAuthorizationHeader"

  predicate {
    type    = "ByteMatch"
    data_id = aws_wafregional_byte_match_set.key_submission_authorization_header.id
    negated = true
  }
  predicate {
    type    = "ByteMatch"
    data_id = aws_wafregional_byte_match_set.key_submission_new_key_claim_uri.id
    negated = false
  }
}

###
# AWS WAF ACL - Key Submission
###

resource "aws_wafregional_web_acl" "key_submission" {
  name        = "KeySubmission"
  metric_name = "KeySubmission"

  default_action {
    type = "ALLOW"
  }

  rule {
    type     = "RATE_BASED"
    priority = 1
    rule_id  = aws_wafregional_rate_based_rule.key_submission_claim_key_uri.id
    action {
      type = "BLOCK"
    }
  }

  rule {
    type     = "REGULAR"
    priority = 2
    rule_id  = aws_wafregional_rule.key_submission_new_key_claim_authorization_header.id
    action {
      type = "BLOCK"
    }
  }
}

resource "aws_wafregional_web_acl_association" "key_submission" {
  resource_arn = aws_lb.covidshield_key_submission.arn
  web_acl_id   = aws_wafregional_web_acl.key_submission.id
}
