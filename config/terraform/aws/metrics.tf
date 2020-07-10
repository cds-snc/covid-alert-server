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
# is also always different, so we need a metric for each possible position it
# can be in. :/


resource "aws_cloudwatch_log_metric_filter" "UnclaimedOneTimeCodeTotal" {
  count = 8

  name           = "UnclaimedOneTimeCodeTotal${count.index}"
  pattern        = "{$.updates[${count.index}].name = \"covidshield.app.unclaimed_one_time_codes.total\"}"
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "CovidShield"
    namespace = "UnclaimedOneTimeCodeTotal"
    value     = "$.updates[${count.index}].sum"
  }
}

resource "aws_cloudwatch_log_metric_filter" "ClaimedOneTimeCodeTotal" {
  count = 8

  name           = "ClaimedOneTimeCodeTotal${count.index}"
  pattern        = "{$.updates[${count.index}].name = \"covidshield.app.claimed_one_time_codes.total\"}"
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "CovidShield"
    namespace = "ClaimedOneTimeCodeTotal"
    value     = "$.updates[${count.index}].sum"
  }
}

resource "aws_cloudwatch_log_metric_filter" "DiagnosisKeyTotal" {
  count = 8

  name           = "DiagnosisKeyTotal${count.index}"
  pattern        = "{$.updates[${count.index}].name = \"covidshield.app.diagnosis_keys.total\"}"
  log_group_name = aws_cloudwatch_log_group.covidshield.name

  metric_transformation {
    name      = "CovidShield"
    namespace = "DiagnosisKeyTotal"
    value     = "$.updates[${count.index}].sum"
  }
}