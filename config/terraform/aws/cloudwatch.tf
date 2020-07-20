resource "aws_cloudwatch_log_group" "covidshield" {
  name       = var.cloudwatch_log_group_name
  kms_key_id = aws_kms_key.cw.arn

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

###
# AWS CloudWatch Metrics - Scaling metrics
###

resource "aws_cloudwatch_metric_alarm" "retrieval_cpu_utilization_high" {
  alarm_name          = "retrieval-cpu-utilization-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = "120"
  statistic           = "Average"
  threshold           = "50"
  alarm_description   = "This metric monitors ecs cpu utilization"

  alarm_actions = [aws_sns_topic.alert_warning.arn]
  dimensions = {
    ClusterName = aws_ecs_cluster.covidshield.name
    ServiceName = aws_ecs_service.covidshield_key_retrieval.name
  }
}

resource "aws_cloudwatch_metric_alarm" "submission_cpu_utilization_high" {
  alarm_name          = "submission-cpu-utilization-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = "120"
  statistic           = "Average"
  threshold           = "50"
  alarm_description   = "This metric monitors ecs cpu utilization"

  alarm_actions = [aws_sns_topic.alert_warning.arn]

  dimensions = {
    ClusterName = aws_ecs_cluster.covidshield.name
    ServiceName = aws_ecs_service.covidshield_key_submission.name
  }
}

resource "aws_cloudwatch_metric_alarm" "retrieval_memory_utilization_high" {
  alarm_name          = "retrieval-memory-utilization-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "MemoryUtilization"
  namespace           = "AWS/ECS"
  period              = "120"
  statistic           = "Average"
  threshold           = "50"
  alarm_description   = "This metric monitors ecs memory utilization"

  alarm_actions = [aws_sns_topic.alert_warning.arn]

  dimensions = {
    ClusterName = aws_ecs_cluster.covidshield.name
    ServiceName = aws_ecs_service.covidshield_key_retrieval.name
  }
}

resource "aws_cloudwatch_metric_alarm" "submission_memory_utilization_high" {
  alarm_name          = "submission-memory-utilization-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "MemoryUtilization"
  namespace           = "AWS/ECS"
  period              = "120"
  statistic           = "Average"
  threshold           = "50"
  alarm_description   = "This metric monitors ecs memory utilization"

  alarm_actions = [aws_sns_topic.alert_warning.arn]

  dimensions = {
    ClusterName = aws_ecs_cluster.covidshield.name
    ServiceName = aws_ecs_service.covidshield_key_submission.name
  }
}

###
# AWS CloudWatch Metrics - Unauthorized requests
###

resource "aws_cloudwatch_log_metric_filter" "unauthorized_new_one_time_code_requests" {
  name           = "UnauthorizedNewOneTimeCodeRequests"
  pattern        = "statusCode=401 msg=\"http response\" \"path=/new-key-claim\""
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "UnauthorizedNewOneTimeCodeRequests"
    namespace = "CovidShield"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "unauthorized_new_one_time_code_requests_warn" {
  alarm_name          = "UnauthorizedNewOneTimeCodeRequestsWarn"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.unauthorized_new_one_time_code_requests.name
  namespace           = "CovidShield"
  period              = "300"
  statistic           = "Sum"
  threshold           = "60"
  alarm_description   = "This metric monitors the 401 response rate for /new-key-claim"

  alarm_actions = [aws_sns_topic.alert_warning.arn]
}

resource "aws_cloudwatch_log_metric_filter" "unauthorized_one_time_code_claim_requests" {
  name           = "UnauthorizedOneTimeCodeClaimRequests"
  pattern        = "statusCode=401 msg=\"http response\" \"path=/claim-key\""
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "UnauthorizedOneTimeCodeClaimRequests"
    namespace = "CovidShield"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "unauthorized_one_time_code_claim_requests_warn" {
  alarm_name          = "UnauthorizedOneTimeCodeClaimRequestsWarn"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.unauthorized_one_time_code_claim_requests.name
  namespace           = "CovidShield"
  period              = "300"
  statistic           = "Sum"
  threshold           = "60"
  alarm_description   = "This metric monitors the 401 response rate for /claim-key"

  alarm_actions = [aws_sns_topic.alert_warning.arn]
}

resource "aws_cloudwatch_log_metric_filter" "unauthorized_upload_requests" {
  name           = "UnauthorizedUploadRequests"
  pattern        = "statusClass=4xx msg=\"http response\" \"path=/upload\""
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "UnauthorizedUploadRequests"
    namespace = "CovidShield"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "unauthorized_upload_requests_warn" {
  alarm_name          = "UnauthorizedUploadRequestsWarn"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.unauthorized_upload_requests.name
  namespace           = "CovidShield"
  period              = "300"
  statistic           = "Sum"
  threshold           = "60"
  alarm_description   = "This metric monitors the 4xx response rate for /upload"

  alarm_actions = [aws_sns_topic.alert_warning.arn]
}

###
# AWS CloudWatch Metrics - Code errors
###

resource "aws_cloudwatch_log_metric_filter" "five_hundred_response" {
  name           = "500Response"
  pattern        = "statusClass=5xx msg=\"http response\""
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "500Response"
    namespace = "CovidShield"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "five_hundred_response_warn" {
  alarm_name          = "500ResponseWarn"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.five_hundred_response.name
  namespace           = "CovidShield"
  period              = "60"
  statistic           = "Sum"
  threshold           = "1"
  alarm_description   = "This metric monitors for an 5xx level response"

  alarm_actions = [aws_sns_topic.alert_warning.arn]
}

resource "aws_cloudwatch_log_metric_filter" "error_logged" {
  name           = "ErrorLogged"
  pattern        = "level=error"
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "ErrorLogged"
    namespace = "CovidShield"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "error_logged_warn" {
  alarm_name          = "ErrorLoggedWarn"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.error_logged.name
  namespace           = "CovidShield"
  period              = "60"
  statistic           = "Sum"
  threshold           = "1"
  alarm_description   = "This metric monitors for an error level logs"

  alarm_actions = [aws_sns_topic.alert_warning.arn]
}

resource "aws_cloudwatch_log_metric_filter" "fatal_logged" {
  name           = "FatalLogged"
  pattern        = "level=fatal"
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "FatalLogged"
    namespace = "CovidShield"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "fatal_logged_warn" {
  alarm_name          = "FatalLoggedWarn"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.fatal_logged.name
  namespace           = "CovidShield"
  period              = "60"
  statistic           = "Sum"
  threshold           = "1"
  alarm_description   = "This metric monitors for a fatal level logs"

  alarm_actions = [aws_sns_topic.alert_warning.arn, aws_sns_topic.alert_critical.arn]
}

###
# AWS CloudWatch Metrics - DDoS Alarms
###

resource "aws_cloudwatch_metric_alarm" "ddos_detected_submission" {
  alarm_name          = "DDoSDetectedSubmissionALB"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "DDoSDetected"
  namespace           = "AWS/DDoSProtection"
  period              = "60"
  statistic           = "Sum"
  threshold           = "1"
  alarm_description   = "This metric monitors for DDoS detected on server ALB"

  alarm_actions = [aws_sns_topic.alert_warning.arn, aws_sns_topic.alert_critical.arn]

  dimensions = {
    ResourceArn = aws_lb.covidshield_key_server.arn
  }
}
