# Why do we have a count of 8 you ask?
#
# App metrics are logged as an array in JSON ex:
#
# {
#   "time": "2020-07-09T22:06:18.913005245Z",
#   "updates": [
#     {
#       "name": "covidshield.system.memory.free",
#       "min": 16060346368,
#       "max": 16060346368,
#       "sum": 16060346368,
#       "count": 1
#     },
#     {
#       "name": "covidshield.system.cpu.percent",
#       "min": 0.5494505494505346,
#       "max": 0.5494505494505346,
#       "sum": 0.5494505494505346,
#       "count": 1
#     },
#    {
#       "name": "covidshield.app.claimed_one_time_codes.total",
#       "min": 403,
#       "max": 403,
#       "sum": 403,
#       "count": 1
#    }
#    ...
# }
#
# The problem is that this array has 8 items and AWS Cloudwatch can't filter on
# a variable position. ex: {$.updates[*].name = ... The order of the array
# is also always different, so we need a metric filter for each possible position
# can be in. :/


resource "aws_cloudwatch_log_metric_filter" "UnclaimedOneTimeCodeTotal" {
  count = 8

  name           = "UnclaimedOneTimeCodeTotal${count.index}"
  pattern        = "{$.updates[${count.index}].name = \"covidshield.app.unclaimed_one_time_codes.total\"}"
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "UnclaimedOneTimeCodeTotal"
    namespace = "CovidShield"
    value     = "$.updates[${count.index}].sum"
  }
}

resource "aws_cloudwatch_metric_alarm" "UnclaimedOneTimeCodeTotalWarn" {
  alarm_name          = "UnclaimedOneTimeCodeTotalWarn"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.UnclaimedOneTimeCodeTotal.name
  namespace           = "CovidShield"
  period              = "60"
  statistic           = "Sum"
  threshold           = "250"
  alarm_description   = "This metric monitors for total unclaimed codes"

  alarm_actions = [aws_sns_topic.alert_warning.arn]
}

resource "aws_cloudwatch_metric_alarm" "UnclaimedOneTimeCodeTotalCritical" {
  alarm_name          = "UnclaimedOneTimeCodeTotalCritical"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.UnclaimedOneTimeCodeTotal.name
  namespace           = "CovidShield"
  period              = "60"
  statistic           = "Sum"
  threshold           = "400"
  alarm_description   = "This metric monitors for total unclaimed codes"

  alarm_actions = [aws_sns_topic.alert_critical.arn]
}

resource "aws_cloudwatch_log_metric_filter" "ClaimedOneTimeCodeTotal" {
  count = 8

  name           = "ClaimedOneTimeCodeTotal${count.index}"
  pattern        = "{$.updates[${count.index}].name = \"covidshield.app.claimed_one_time_codes.total\"}"
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "ClaimedOneTimeCodeTotal"
    namespace = "CovidShield"
    value     = "$.updates[${count.index}].sum"
  }
}

resource "aws_cloudwatch_metric_alarm" "ClaimedOneTimeCodeTotalWarn" {
  alarm_name          = "ClaimedOneTimeCodeTotalWarn"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.ClaimedOneTimeCodeTotal.name
  namespace           = "CovidShield"
  period              = "60"
  statistic           = "Sum"
  threshold           = "3000"
  alarm_description   = "This metric monitors for total claimed codes"

  alarm_actions = [aws_sns_topic.alert_warning.arn]
}

resource "aws_cloudwatch_metric_alarm" "ClaimedOneTimeCodeTotalCritical" {
  alarm_name          = "ClaimedOneTimeCodeTotalCritical"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.ClaimedOneTimeCodeTotal.name
  namespace           = "CovidShield"
  period              = "60"
  statistic           = "Sum"
  threshold           = "4500"
  alarm_description   = "This metric monitors for total claimed codes"

  alarm_actions = [aws_sns_topic.alert_critical.arn]
}

resource "aws_cloudwatch_log_metric_filter" "DiagnosisKeyTotal" {
  count = 8

  name           = "DiagnosisKeyTotal${count.index}"
  pattern        = "{$.updates[${count.index}].name = \"covidshield.app.diagnosis_keys.total\"}"
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "DiagnosisKeyTotal"
    namespace = "CovidShield"
    value     = "$.updates[${count.index}].sum"
  }
}

resource "aws_cloudwatch_metric_alarm" "DiagnosisKeyTotalWarn" {
  alarm_name          = "DiagnosisKeyTotalWarn"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.DiagnosisKeyTotal.name
  namespace           = "CovidShield"
  period              = "60"
  statistic           = "Sum"
  threshold           = "9000"
  alarm_description   = "This metric monitors for total diagnosis keys"

  alarm_actions = [aws_sns_topic.alert_warning.arn]
}

resource "aws_cloudwatch_metric_alarm" "DiagnosisKeyTotalCritical" {
  alarm_name          = "DiagnosisKeyTotalCritical"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.DiagnosisKeyTotal.name
  namespace           = "CovidShield"
  period              = "60"
  statistic           = "Sum"
  threshold           = "13500"
  alarm_description   = "This metric monitors for total diagnosis keys"

  alarm_actions = [aws_sns_topic.alert_critical.arn]
}